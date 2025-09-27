# mkawanoメモ

## 初期セットアップ
- `./init.sh` でリポジトリの初期化
- VM環境: `./init.sh`
- ローカル環境: `./init.sh [VMのパブリックIP] [秘密鍵のパス]`

1. 重複実行防止: .da/.initLockファイルで1回のみ実行可能
2. 環境判定: ホスト名でVM環境かローカル環境かを判別
   - VM環境（ftt2508-*）: データを直接コピー
   - ローカル環境: VM IPアドレスと秘密鍵を引数で指定
3. データダウンロード: GitHubリリースから初期データ（restoreSQL.zip）をダウンロード
4. Azure Container Registry: Dockerトークンでログイン
5. データベース復元: restore_and_migration.shを実行
6. 成功時: ロックファイル作成、Webアクセス先とテスト実行方法を表示

### `restore_and_migration.sh`
1. コンテナ再起動: restart_container.shを実行
2. DB再作成: 42tokyo2508-dbデータベースを削除・再作成
3. 初期化: init.sqlを実行
4. データリストア: 環境・引数に応じてSQLファイルを選択
   - 引数あり（e2e）: e2e_users.sql, e2e_products.sql + 指定ファイル
   - VM環境: remote_all.sql
   - ローカル環境: local_all.sql
5. マイグレーション: mysql/migration/内の0_*.sql, 1_*.sql...を順次実行

## 全体像
```mermaid
graph TD
    A[スコア3の原因特定] --> B[Frontend外の全層最適化]
    B --> C[MySQL層]
    B --> D[Backend層] 
    B --> E[nginx層]
    C --> F[クエリ最適化・インデックス]
    D --> G[並行処理・メモリ最適化]
    E --> H[キャッシュ・圧縮・接続プール]
```

## 調査・分析

## 改善案

## TODO

## その他