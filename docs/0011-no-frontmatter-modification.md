# ADR 0011: skraft は SKILL.md の frontmatter を編集しない

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ
- **関連**: ADR 0002 (SQLite ledger), ADR 0009 (バージョニング戦略)

## 文脈

skraft は SKILL.md を扱う CLI ツールである。設計を進めるうえで、「skraft は SKILL.md の frontmatter に何を書き込むべきか」を決める必要があった。

skraft が書き込むことを検討した情報を整理すると:

| 情報 | 用途 |
|---|---|
| `created_at` | スキル作成日時の記録 |
| `version` | スキルのバージョン |
| `stage` | ライフサイクルステージ (draft / beta / stable など) |
| アップロード状態 | Claude.ai へのアップロード履歴 |
| メンテナ情報 | 担当者の連絡先 |

しかし、これらすべてに**より良い情報源が既に存在する**ことが分かった:

| 情報 | 代替情報源 |
|---|---|
| `created_at` | Git log (最初のコミット日時) |
| `updated_at` | Git log (最新のコミット日時) |
| `version` | Git tag (ADR 0009) |
| Claude.ai へのアップロード状態 | SQLite ledger (ADR 0002) |
| Claude Code との link 状態 | ファイルシステム (symlink の存在) |
| ステージ遷移履歴 (将来) | SQLite ledger |

つまり、**skraft が SKILL.md に書き込む必要のある情報は存在しない**。すべての状態は Git、ファイルシステム、SQLite ledger のいずれかで管理できる。

加えて、SKILL.md は skraft 専用ではなく、複数のツールが frontmatter を扱う:

- skill-creator が `name` と `description` を生成
- `gh skill install` が provenance metadata (ソースリポジトリ、ref、tree SHA) を書き込む
- Agent Skills 仕様が `name`, `description`, `license`, `allowed-tools` を予約フィールドとして定義

skraft が frontmatter に書き込むと、これらのツールとの相互運用で衝突リスクが生じる。

## 検討した選択肢

| 候補 | 概要 | 評価 |
|---|---|---|
| A. `lifecycle:` ネスト | `lifecycle.created_at` のような構造で skraft 固有情報をまとめる | 衝突回避できるが、書き込みアルゴリズム・名前空間予約のコストが発生 |
| B. フラット + プレフィックス | `skraft_created_at:` のように書く | 衝突リスク低、冗長 |
| C. フラット + 一般名 | `created_at:` をトップレベルに書く | 衝突リスク高 (gh skill 等が将来同名を使う可能性) |
| **D. frontmatter を編集しない** | skraft は読むだけ、書き込まない | **責務が純粋になる、設計が単純化** |

## 決定

**skraft は SKILL.md の frontmatter を編集しない (案 D)。**

skraft は SKILL.md を**読み取り専用**で扱う。`name` や `description` を必要に応じて読み取るが、frontmatter には一切の書き込みを行わない。

skraft 固有の状態 (Claude.ai アップロード状態など) は SQLite ledger で管理する。バージョン情報は git tag から動的に取得する。

## 理由

### 1. skraft の責務が純粋になる

skraft の責務は「環境間の状態同期」と「Claude.ai アップロード状態管理」である。SKILL.md の内容そのものはユーザーや skill-creator が管理する領域であり、skraft が書き換えるのは責務逸脱に近い。

「skraft は SKILL.md を**観察するツール**であって、**変更するツールではない**」という位置づけが、責務分離の原則と整合する。

### 2. 設計が大幅に単純化される

frontmatter を編集しないことで、以下の複雑性がすべて回避される:

- 文字列ベース編集アルゴリズム (コメント・空白・順序の保持)
- パース/再シリアライズの整合性チェック
- 名前空間の予約と衝突回避
- 編集後の YAML valid 性検証
- skill-creator や gh skill との書き込み境界の交渉

### 3. ファイル破壊リスクがゼロになる

skraft が SKILL.md を書き換えなければ、**skraft が原因でユーザーのスキル定義が壊れる事故は原理的に起こらない**。これは個人開発ツールにとって重要な性質。「触らないものは壊さない」は、信頼を獲得する最短経路である。

### 4. 他ツールとの相互運用が完璧になる

skill-creator、gh skill、その他のツールが何を書き込んでも、skraft はそれらと衝突しない。`gh skill install` が新しい provenance metadata フィールドを増やしても、skill-creator が新しい慣習を導入しても、skraft は影響を受けない。

### 5. ユーザー体験がシンプルになる

`skraft init` を実行しても SKILL.md の内容が変わらない。ユーザーは「skraft が何を書き込んだか」を意識する必要がなく、安心して使える。`git diff` がノイズだらけになる事故も起きない。

### 6. 既存情報源を活用するという ADR 0009 の論理の延長

ADR 0009 で「git tag を正本にして frontmatter に version を書かない」と決めた。本 ADR はその論理を `created_at` などの他の情報にも適用したものである。**Git や ledger に既にある情報を、skraft が再保存しない**という方針で一貫している。

## 検討したが採用しなかった選択肢

### 案 A (`lifecycle:` ネスト) を採用しなかった理由

`lifecycle.created_at` のような構造で skraft 固有情報を 1 つの名前空間にまとめる案。衝突回避としては有効だが、そもそも書き込む情報が代替情報源で取得できる以上、名前空間を予約する必要がない。書き込みアルゴリズムや予約領域の運用ルールを設けるオーバーヘッドが、得られる利益より大きい。

### 案 B (`skraft_created_at:` プレフィックス) を採用しなかった理由

衝突リスクは低いが、名前が冗長で読みにくい。書き込む必要がないという結論に至った時点で、名前空間設計の議論自体が不要になった。

### 案 C (`created_at:` フラット) を採用しなかった理由

`created_at` という名前は一般的すぎて、gh skill や他ツールが将来同名のフィールドを書き始める可能性が高い。衝突した場合に skraft の動作が不定になる。

## 結果

### `skraft init` の挙動

`skraft init` は SKILL.md には触らない。代わりに以下を作成する:

```
my-skills/
├── .skraft/
│   ├── config.toml      ← skraft 設定
│   └── ledger.db        ← SQLite ledger
└── .gitignore           ← .skraft/ledger.db* を追記
```

各スキルディレクトリの SKILL.md は**一切変更されない**。これにより、`skraft init` は idempotent な操作になり、何度実行しても同じ結果になる。

### `created_at` を MVP では使わない

`created_at` は skraft の MVP 機能 (`status`, `pack`, `link`, `sync`, `mark-uploaded`) のいずれにも必須ではない。MVP では `created_at` を取得・表示しない。

将来 `created_at` が必要になった場合は、Git log から動的に取得する設計を別 ADR で議論する。

### `skraft status` の表示

skraft が独自に管理する状態 (Claude.ai アップロード状態など) と、Git から取得する情報 (tag、HEAD) のみを表示する:

```
$ skraft status
Repository: my-skills/
Latest tag: v1.2.0 (a1b2c3d, 5 days ago)
HEAD:       abc123def (3 commits ahead of v1.2.0)
State:      unreleased changes

Skills:
  skill-a    SKILL.md exists, linked to Claude Code
  skill-b    SKILL.md exists, last uploaded to Claude.ai at v1.1.0 ⚠ outdated
  skill-c    SKILL.md exists, never uploaded to Claude.ai
```

各情報の取得元:

| 情報 | 取得元 |
|---|---|
| `Latest tag`, `HEAD` | Git |
| `SKILL.md exists` | ファイルシステム |
| `linked to Claude Code` | シンボリックリンクの存在確認 |
| `last uploaded to Claude.ai` | SQLite ledger (`upload_state` テーブル) |

### skraft が SKILL.md に対して行う唯一の操作

skraft は SKILL.md に対して以下のみを行う:

1. **読む** — `name`, `description` などを必要に応じて読み取る (`status` 表示など)
2. **存在確認する** — `os.Stat` で SKILL.md がディスク上に存在するかチェック
3. **コピーする** — `skraft pack` 時に zip に含めるためにバイト単位でコピー
4. **シンボリックリンクを張る** — `skraft link` 時に `~/.claude/skills/` から symlink

これら 4 つはすべて**読み取り専用**または**メタデータ操作 (パーミッション変更ではなく symlink 作成)** であり、SKILL.md の内容を変更しない。

## 解決すべき課題

### 将来 frontmatter に書きたい情報が出てきたら?

その時点で、本 ADR を改訂する別 ADR を立て、書き込みの必要性と方針を議論する。「絶対に書かない」とは言わず、「**MVP では書かない**」という方針として捉える。

ただし新規の書き込み導入は慎重に検討すべき。書き始めると、編集アルゴリズムや名前空間予約の議論が必要になる。それを避けるためにも、書き込みは可能な限り回避する。

### 他ツールが lifecycle: を書き始めたら?

`gh skill` や `skill-creator` が `lifecycle:` というキーを使い始める可能性は否定できない。skraft はそれを読むことはあるが、書き換えない。skraft の挙動には影響しない。

## 関連 ADR

- ADR 0002: SQLite を ledger のストレージとして採用 (skraft 固有の状態を ledger で管理)
- ADR 0009: バージョニング戦略を git tag 中心に置く (`version` を frontmatter に書かない判断の延長)

## 参考

- [YAGNI 原則: https://martinfowler.com/bliki/Yagni.html]
- [Agent Skills 仕様: https://agentskills.io/specification]
