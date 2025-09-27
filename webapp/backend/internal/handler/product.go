package handler

import (
	"backend/internal/middleware"
	"backend/internal/model"
	"backend/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProductHandler struct {
	ProductSvc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{ProductSvc: svc}
}

// 商品一覧を取得
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req model.ListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.SortField == "" {
		req.SortField = "product_id"
	}
	if req.SortOrder == "" {
		req.SortOrder = "asc"
	}
	req.Offset = (req.Page - 1) * req.PageSize

	products, total, err := h.ProductSvc.FetchProducts(r.Context(), userID, req)
	if err != nil {
		log.Printf("Failed to fetch products for user %d: %v", userID, err)
		http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Data  []model.Product `json:"data"`
		Total int             `json:"total"`
	}{
		Data:  products,
		Total: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 注文を作成
func (h *ProductHandler) CreateOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	insertedOrderIDs, err := h.ProductSvc.CreateOrders(r.Context(), userID, req.Items)
	if err != nil {
		log.Printf("Failed to create orders: %v", err)
		http.Error(w, "Failed to process order request", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "Orders created successfully",
		"order_ids": insertedOrderIDs,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ProductHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	imagePath := r.URL.Query().Get("path")
	if imagePath == "" {
		http.Error(w, "画像パスが指定されていません", http.StatusBadRequest)
		return
	}

	imagePath = filepath.Clean(imagePath)
	if filepath.IsAbs(imagePath) || strings.Contains(imagePath, "..") {
		http.Error(w, "無効なパスです", http.StatusBadRequest)
		return
	}

	baseImageDir := "/app/images"
	fullPath := filepath.Join(baseImageDir, imagePath)

	fileInfo, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.Error(w, "画像が見つかりません", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "ファイル情報の取得に失敗しました", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(fullPath)
	var contentType string
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
	}

	// キャッシュヘッダーの設定
	modTime := fileInfo.ModTime()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1時間キャッシュ
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, fileInfo.Size(), modTime.Unix()))

	// If-Modified-Sinceヘッダーのチェック
	if ifModSince := r.Header.Get("If-Modified-Since"); ifModSince != "" {
		if t, err := http.ParseTime(ifModSince); err == nil && !modTime.After(t.Add(1*time.Second)) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// If-None-Matchヘッダーのチェック
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		etag := fmt.Sprintf(`"%x-%x"`, fileInfo.Size(), modTime.Unix())
		if ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// ストリーミング配信（メモリ効率的）
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, filepath.Base(fullPath), modTime, file)
}
