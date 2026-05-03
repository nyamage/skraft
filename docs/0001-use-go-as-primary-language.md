# ADR 0001: Go を主要言語として採用する

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ

## 文脈

skraft は Claude のスキル (Agent Skills) を Git リポジトリで管理し、Claude Code と Claude.ai の間で同期させる CLI ツールである。MVP 段階で実装言語を決める必要があり、長期保守と配布性を考慮した選定を行う。

skraft の特性から、言語選定で重視する判断軸は以下:

- **配布の容易さ**: 個人開発者がインストールしやすいか
- **CLI 体験の質**: サブコマンド、help、進捗表示が作りやすいか
- **ファイルシステム操作**: symlink、zip、YAML、SQLite が素直に扱えるか
- **サブプロセス管理**: Promptfoo などの外部ツール呼び出しが堅実にできるか
- **エコシステム整合性**: `gh skill` など隣接ツールと並べた時の自然さ
- **保守性**: 1 人で長期保守できるか

## 検討した選択肢

| 候補 | 主なメリット | 主なデメリット |
|---|---|---|
| **Go** | 単一バイナリ、`go install` 即配布、Cobra 成熟、`gh` と整合 | YAML のコメント保持が標準ライブラリで弱い |
| Python | YAML / SQLite が標準で扱いやすい、MVP までが速い | 配布が `pip` / `pipx` 前提になる、起動が遅い |
| Rust | 配布性とパフォーマンスに優れる | 学習コストが高い、開発サイクルが遅い |
| TypeScript (Bun) | npm エコシステム、Anthropic SDK と親和的 | グローバルインストールの慣習が CLI と相性悪 |
| Bash | 依存ゼロ、移植性が高い | SQLite と YAML の扱いが現実的でない、状態管理が破綻する |

## 決定

**Go を採用する。**

## 理由

### 1. 単一バイナリで配布できる

`go install github.com/<owner>/skraft@latest` の 1 コマンドでインストール可能。Homebrew、apt、winget への展開も容易。Python 環境を前提にせずに済むため、ユーザー層が広がる。

### 2. `gh skill` と同じエコシステムに乗る

skraft の責務分離図において `gh skill` は配布層を担い、skraft は状態管理層を担う。両者を同じ言語で書くことで、将来 `gh extension` として `gh-skraft` 形式で配布する選択肢が残る。

### 3. サブプロセス管理が安全

Promptfoo などの外部 CLI を呼び出す際、`context.Context` によるタイムアウトとキャンセル制御が標準で備わる。CI で並列実行する用途で特に有利。

### 4. 並行処理が自然

`sync --check` は Git、Claude Code、Claude.ai の 3 環境を比較する。これを goroutine で並列化することで、I/O を待たずに状態取得を完了できる。

### 5. クロスコンパイルが標準サポート

macOS、Linux、Windows のバイナリを 1 つのプロジェクトから出せる。GoReleaser を使えばリリースが自動化できる。

## 検討したが採用しなかった選択肢

### Python を採用しなかった理由

YAML 操作 (frontmatter の往復) と MVP までの速度では Python が優位。しかし配布が `pipx install` 前提になり、ユーザー層が Python 開発者に偏る。skraft は Claude スキルを使うすべての開発者 (Web、モバイル、ML、データ職) を対象とするため、言語非依存な配布が可能な Go を優先した。

### Rust を採用しなかった理由

配布性とパフォーマンスは Go と同等以上。しかし MVP 段階での開発速度を優先したため、所有権・ライフタイムの学習コストを背負う必要のない Go を選んだ。Star が増えて本格運用フェーズに入った段階で、ホットパスを Rust に書き直す選択肢は残る。

### TypeScript を採用しなかった理由

Anthropic 公式 SDK が TypeScript を提供しており、エコシステム親和性は高い。しかし Node.js の起動オーバーヘッドが CLI 用途では重く、`npm install -g` の慣習がグローバル汚染を引き起こす。Bun を使えば改善するが、ランタイムを限定する制約が増える。

### Bash を採用しなかった理由

`init` と `link` 程度なら成立する。しかし `sync --check` の状態比較ロジックや、SQLite ベースの ledger を扱うと破綻する。skraft の規模では不適。

## 結果

### 採用するライブラリ

```go
require (
    github.com/spf13/cobra        // CLI フレームワーク
    github.com/spf13/viper        // 設定管理
    github.com/goccy/go-yaml      // YAML (コメント保持を意識)
    modernc.org/sqlite            // SQLite (CGO 不要)
    github.com/fatih/color        // 色付き出力
)
```

### 解決すべき課題

#### YAML frontmatter のコメント保持

Go の標準的な YAML ライブラリは、コメントや空白の保持が Python の `ruamel.yaml` ほど直感的ではない。SKILL.md には既存のコメントが含まれることが想定されるため、これを破壊しない実装が必要。

**対応方針**: SKILL.md 全体を YAML パースして再シリアライズせず、`lifecycle:` セクションのみ文字列ベースで挿入・置換する。frontmatter の他の部分には触らない。詳細は別 ADR (`0003-string-based-frontmatter-editing.md`) で記述する。

#### SQLite の CGO 依存回避

`mattn/go-sqlite3` は CGO 依存でクロスコンパイルが煩雑になる。これを避けるため、純 Go 実装の `modernc.org/sqlite` を採用する。パフォーマンスは劣るが、skraft の用途 (ローカル ledger、書き込み頻度低) では問題にならない。

### 配布計画

- **第 1 段階**: `go install` での配布
- **第 2 段階**: GitHub Releases にプリビルドバイナリを置く (GoReleaser で自動化)
- **第 3 段階**: Homebrew formula (`brew install skraft`)
- **第 4 段階**: apt / winget / Scoop パッケージ

## 関連 ADR

- ADR 0002: SQLite を ledger のストレージとして採用
- ADR 0011: skraft は SKILL.md frontmatter を編集しない

## 参考

- [Go によるモダン CLI 設計の事例: gh, terraform, kubectl]
- [GoReleaser: https://goreleaser.com/]
- [modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite]
