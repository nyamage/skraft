# ADR 0002: SQLite を ledger のストレージとして採用する

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ
- **関連**: ADR 0001 (Go を採用)

## 文脈

skraft は frontmatter で「現在の状態」を、**ledger** で「過去に起きたすべて」を管理する。frontmatter は YAML として SKILL.md に直接書き込まれるが、ledger は時系列の出来事 (eval 結果、invocation、Claude.ai へのアップロード記録、ステージ遷移など) を蓄積するため、別のストレージが必要になる。

ledger に求められる要件:

- **時系列クエリ**: 「過去 30 日間に X 回以上発火したスキル」を取り出せる
- **集計クエリ**: `status` や `history` コマンドで複数イベントを集計表示
- **冪等な書き込み**: OTEL イベントを再取り込みしても二重登録されない
- **並行アクセス安全**: ローカル CLI と CI が同時に書く可能性がある
- **単一ファイル**: バックアップ、移動、共有が楽
- **言語非依存**: 将来 skraft を Rust などで書き直しても、ledger を再利用できる
- **依存ゼロ寄り**: 別途サーバーを立てない

ledger は MVP では使われないが、将来の `health` / `history` / `report` コマンドの基盤となる。データモデルを最初に固めておかないと、後から差し替えるコストが高い。

## 検討した選択肢

| 候補 | 主なメリット | 主なデメリット |
|---|---|---|
| **SQLite** | SQL クエリ可能、単一ファイル、並行安全、言語非依存 | スキーママイグレーションを自分で管理する必要 |
| JSONL (append-only) | シンプル、テキストで grep 可能、append が安全 | ランダムアクセス不可、集計が遅い、削除が困難 |
| JSON ファイル群 | デバッグが楽、git diff が読める | ファイル数増加でパフォーマンス低下、トランザクション性なし |
| BoltDB / BadgerDB | Go ネイティブ、組み込み KV ストア | SQL クエリ不可、デバッグツールが SQLite ほど豊富でない |
| TOML / YAML | 人間可読、設定向きでよく使われる | 時系列データに不向き、追記が非効率 |
| 外部 DB (PostgreSQL 等) | 強力なクエリ、本格的な並行制御 | 個人開発で過剰、サーバー運用が必要 |

## 決定

**SQLite を ledger のストレージとして採用する。**

データベースファイルは `.skraft/ledger.db` に配置する。

## 理由

### 1. SQL クエリで集計が直感的に書ける

`history` や `health` コマンドは複雑な集計を必要とする。たとえば「過去 7 日間で false positive 率がしきい値を超えたスキル」のような問い合わせを、SQL なら 1 行で書ける。JSONL や KV ストアでは、これを毎回コードで実装する必要がある。

### 2. 単一ファイルでの取り回しが楽

ledger.db は SQLite のディスクフォーマット仕様で 1 ファイルにまとまる。バックアップは `cp` で済み、別マシンへの移動も同様。チームで共有する場合も Git LFS で 1 ファイル管理できる (推奨はしないが、選択肢として残る)。

### 3. 並行アクセスが安全

SQLite は WAL (Write-Ahead Logging) モードで複数のリーダーと 1 つのライターを同時に扱える。skraft の利用パターンでは、ローカル CLI と CI が同時に書くケースが想定されるが、ファイルロックで衝突を防げる。

### 4. 言語非依存

将来 skraft を Rust や別言語で書き直しても、ledger.db はそのまま使える。SQLite のフォーマットは公開仕様で、20 年以上の互換性が保証されている。

### 5. デバッグツールが豊富

`sqlite3 .skraft/ledger.db` で対話的にクエリできる。GUI も `DB Browser for SQLite` などが揃っており、ユーザーが直接データを覗ける。これは KV ストアにない大きな利点。

### 6. modernc.org/sqlite で CGO 不要

純 Go 実装の SQLite ドライバを使うことで、クロスコンパイルが容易になる。ADR 0001 で議論した配布性の利点を損なわない。

## 検討したが採用しなかった選択肢

### JSONL を採用しなかった理由

append-only ログとしてはシンプルで、OTEL イベントの取り込みには合う。しかし `history` コマンドのような集計クエリで、毎回ファイル全体を読み直す必要がありパフォーマンスが劣化する。**ただし**、SQLite を主ストレージとしつつ、生イベントを JSONL で並行保存する選択肢は残す (デバッグや再構築のため)。

### JSON ファイル群を採用しなかった理由

スキルごとに `events/<skill>/2026-05-03.json` のように分割する案も考えた。デバッグはしやすいが、ファイル数が線形に増えて IO 負荷が高くなる。トランザクション性も弱く、書き込み中のクラッシュでファイルが壊れるリスクがある。

### BoltDB / BadgerDB を採用しなかった理由

Go ネイティブで CGO 不要、シンプルな KV ストアとしては魅力的。しかし KV モデルでは集計クエリを書く際に毎回キー走査が必要で、SQL ほど直感的に書けない。デバッグツールも SQLite ほど豊富ではなく、ユーザーが直接覗く際の体験が劣る。

### 外部 DB (PostgreSQL 等) を採用しなかった理由

クエリ性能と並行制御は最も強力だが、個人開発の CLI ツールでサーバーを要求するのは過剰。skraft は「単一ユーザーの単一マシン」を主な利用シーンとし、ledger を**ローカルの単一ファイル**として持つ思想を維持する。

## 結果

### スキーマ案

最初に作る主要テーブル:

```sql
-- イベント (時系列)
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    otel_event_id TEXT UNIQUE,           -- OTEL の id (冪等性確保)
    timestamp TEXT NOT NULL,             -- ISO 8601
    skill_name TEXT NOT NULL,
    event_type TEXT NOT NULL,            -- 'created' | 'eval' | 'invocation' | 'upload' | 'promotion'
    payload TEXT NOT NULL                -- JSON
);

CREATE INDEX idx_events_skill ON events(skill_name);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_type ON events(event_type);

-- アップロード状態 (Claude.ai への最終アップロード状態を記録)
CREATE TABLE upload_state (
    skill_name TEXT NOT NULL,
    target TEXT NOT NULL,                -- 'claudeai' | 'claude_code'
    version TEXT NOT NULL,
    content_hash TEXT NOT NULL,          -- SHA256 of zip content
    uploaded_at TEXT NOT NULL,
    PRIMARY KEY (skill_name, target)
);

-- メタデータ (スキーマバージョンなど)
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

### 冪等性の確保

OTEL イベントは `otel_event_id` (UUID) を持つため、`INSERT OR IGNORE` で二重登録を防ぐ:

```sql
INSERT OR IGNORE INTO events (otel_event_id, timestamp, skill_name, event_type, payload)
VALUES (?, ?, ?, ?, ?);
```

これにより `skraft ingest` を何度実行しても安全。

### マイグレーション戦略

`metadata` テーブルに `schema_version` を保持し、起動時にバージョンを比較してマイグレーションを実行する。マイグレーションスクリプトは Go の `embed` で埋め込み、別ファイル管理にしない:

```
internal/migrations/
├── 0001_initial.sql
├── 0002_add_upload_state.sql
└── ...
```

ライブラリは `golang-migrate/migrate` または自作の薄いマイグレータを使用。skraft の規模なら自作で十分。

### Git 管理方針

ledger.db は **`.gitignore` に入れる**。理由:

- バイナリファイルなので diff が見えない
- 個人ごとの利用履歴は本来共有するものではない
- マシン間で同期したい場合は Dropbox / Syncthing などで対応する選択肢が残る

ただし、チーム運用では `ledger.db` を意図的にコミットする選択肢もある。これは将来 ADR で別途議論する。

### バックアップ

`skraft backup` のようなコマンドは v1 では作らない。ユーザーは `cp .skraft/ledger.db ledger.db.bak` で十分対応できる。

### パフォーマンス想定

ledger は個人開発の利用パターンで、1 日数十イベント程度を想定。1 年間で 1 万〜10 万行に収まる。SQLite はこの規模で何ら問題なく動く (本来は数億行まで耐える)。インデックス設計を最初から入れることで、`status` や `history` のクエリは数 ms で完了する。

### 解決すべき課題

#### WAL モードの有効化

並行アクセス安全性のため、初回接続時に `PRAGMA journal_mode = WAL;` を実行する。これにより `.skraft/` 以下に `ledger.db-shm` と `ledger.db-wal` も作られる。これらも `.gitignore` に含める。

#### スキーマ変更時の安全策

マイグレーション失敗時に元の DB を破壊しないよう、`skraft init` 後の最初のマイグレーション以外では、実行前に `ledger.db` のスナップショットを `.skraft/backups/` に保存する。

## 関連 ADR

- ADR 0001: Go を主要言語として採用 (modernc.org/sqlite を使う背景)
- ADR 0009: バージョニング戦略を git tag 中心に置く (upload_state.version の意味)
- ADR 0011: skraft は SKILL.md frontmatter を編集しない (skraft 固有の状態は ledger で管理)

## 参考

- [SQLite WAL モード: https://www.sqlite.org/wal.html]
- [SQLite データ型と制限: https://www.sqlite.org/limits.html]
- [modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite]
- [golang-migrate/migrate: https://github.com/golang-migrate/migrate]
