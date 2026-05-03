# ADR 0009: バージョニング戦略を git tag 中心に置く

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ
- **関連**: ADR 0001 (Go), ADR 0002 (SQLite ledger), ADR 0003 (frontmatter 編集)

## 文脈

skraft は MVP 設計の初期段階で、`lifecycle.version: 1.2.0` のように frontmatter にバージョン文字列を書き込む案を採用していた。しかし `gh skill` の仕様と Agent Skills エコシステムの実態を精査した結果、以下の事実が明らかになった:

### Agent Skills 仕様とエコシステムの実態

- `gh skill install owner/repo skill@v1.2.0` のように **git tag をバージョンとして指定する**設計が確立している
- インストール時に SKILL.md の frontmatter には `repository`、`ref`、`tree_sha` が書き込まれ、**`version` フィールドは標準仕様として存在しない**
- `gh skill update` は git tree SHA をローカルとリモートで比較し、**コンテンツアドレス型で変更検知**する
- Agent Skills 仕様 (agentskills.io) でも `version` フィールドは規定されていない (`name`, `description`, `license`, `allowed-tools` のみ)

### モノレポが事実上の標準

主要なスキル配布リポジトリはすべて**モノレポ**である:

- `anthropics/skills` (Anthropic 公式)
- `github/awesome-copilot` (GitHub コミュニティ)
- `openai/skills` (OpenAI 公式)
- `huggingface/skills`
- `vercel-labs/skills`

つまり「**1 リポジトリ = 複数スキル**」が Agent Skills エコシステムの前提となっている。

### `gh skill publish` のデフォルト

`gh skill publish` の対話的フローはリポジトリ全体に 1 つの semver tag を切る形式 (`v1.0.0` など)。これは「**リポジトリ全体を同期リリース**」するモデルを推奨している。

### この状況で skraft が `lifecycle.version` を持つ問題

- **正本の二重化**: git tag と frontmatter の両方にバージョン情報がある
- **整合性管理コスト**: ずれた時にどちらを優先するかの判断が必要
- **gh skill とのモデル不一致**: gh skill は tag 単位で考えるが、skraft は frontmatter 単位で考える

skraft の責務分離方針 (各専門ツールに委譲、状態管理だけを担う) からして、**Git が既に持っているバージョニング機能を再実装する**のは責務違反である。

## 検討した選択肢

### バージョニングの正本

| 候補 | 概要 | 主な評価 |
|---|---|---|
| **A. git tag 中心、frontmatter から version 削除** | git tag を正本とし、skraft は `git describe` で動的取得 | 二重化なし、gh skill 整合 |
| B. frontmatter と git tag を併用、skraft が整合チェック | 両方持ち、ずれを検知 | 状態を独立保持できるが管理コスト大 |
| C. frontmatter のみ、git tag は無視 | skraft 独自管理 | gh skill とずれる、Git 既存機能の無視 |
| D. Semantic Versioning + 自動 tag 生成 | skraft が tag を切る | 越権行為、Git 操作はユーザーの責務 |

### モノレポでのリリース戦略

| 候補 | 概要 | 主な評価 |
|---|---|---|
| **α. 同期リリース** | リポジトリ全体に 1 つの tag (`v1.2.0`) | gh skill デフォルト、エコシステム標準 |
| β. 個別バージョニング (prefix tag) | スキルごとに `skill-a/v1.2.0` のような tag | スキル単位で独立、ツール対応コスト |
| γ. リポジトリ分割 | スキルごとに別リポジトリ | 完全独立だが Git オーバーヘッド大 |

## 決定

**バージョニングの正本: 案 A (git tag 中心、frontmatter から version 削除)**

`lifecycle.version` フィールドは frontmatter から削除する。skraft はバージョン情報を `git describe --tags --abbrev=0` 等で動的に取得する。

**モノレポ戦略: 案 α (同期リリース)**

skraft 管理下のリポジトリは複数スキルを含むモノレポを前提とし、**リポジトリ全体に 1 つの tag** を切る同期リリースをデフォルトとする。

スキルごとの個別バージョニングが必要な場合は、**リポジトリを分割する** (案 γ) ことを推奨する。MVP 時点では prefix tag (案 β) はサポートしない。

## 理由

### 1. 正本が 1 つに集約される

git tag のみがバージョンの正本となる。skraft が別途 frontmatter に書く必要はなく、ずれが原理的に発生しない。

### 2. gh skill モデルと完全に整合する

`gh skill install owner/repo skill@v1.2.0` は git tag を直接指定する。`gh skill publish` も同期リリースをデフォルトとする。skraft が同じモデルで動くことで、両者の相互運用が自然になる。

### 3. Agent Skills エコシステムの標準に従う

主要な公式・コミュニティリポジトリがすべてモノレポ + 同期リリースを採用している。skraft が異なる前提を持つと、ユーザーが他のリポジトリと同じやり方で扱えなくなる。

### 4. Git の既存機能を尊重する

Git は 20 年以上にわたり tag によるバージョニングを提供してきた。skraft が独自の version フィールドや prefix tag 管理を持つのは、**既に解決済みの問題を再実装する**のと同義。

### 5. immutable releases との親和性

`gh skill` は GitHub の **immutable releases** (タグ保護による不変リリース) との連携を推奨している。skraft が tag ベースで動くことで、この保証をそのまま受け継げる。

### 6. ユーザーが手動で frontmatter を更新する手間がなくなる

リリースは **`git tag v1.2.0 && git push --tags`** だけで完了する。skraft が frontmatter を書き換える必要がない。

### 7. シンプルさが MVP に適合する

prefix tag (`skill-a/v1.2.0`) を MVP からサポートすると、`git describe` の `--match` オプションや、スキル単位の HEAD 比較ロジックなど、実装が複雑化する。同期リリースに絞ることで、MVP の実装範囲が明確になる。

## 検討したが採用しなかった選択肢

### 案 B (併用 + 整合チェック) を採用しなかった理由

二重化を許容しつつ skraft が整合をチェックする案。理屈は通るが、「整合させる」こと自体に価値がない。skraft の責務は環境間の状態同期 (Git ↔ Claude Code ↔ Claude.ai) であり、**Git 内部の二重化を解消する責務まで背負うべきではない**。

### 案 C (frontmatter のみ) を採用しなかった理由

gh skill とのモデル不一致が決定的に問題。skraft 単体で完結するなら成立するが、実際には gh skill と並ぶ層として動くため、tag を無視する設計は採れない。

### 案 D (skraft が tag を切る) を採用しなかった理由

skraft が `git tag` を打つのは**ユーザーの Git ワークフローへの越権**。リリース判断はユーザーの責任であり、skraft は状態管理だけに留まるべき。`skraft release` のようなコマンドを将来作るとしても、それは skraft が tag を切るのではなく、**ユーザーに tag を切るよう促す**形に留める。

### 案 β (prefix tag) を MVP で採用しなかった理由

`skill-a/v1.2.0` のような prefix tag は、スキルごとの独立リリースを可能にする。しかし以下の問題がある:

- gh skill の `--pin v1.2.0` のような単純な指定がしづらくなる
- `gh skill publish` の対話的フローと整合しない
- monotag のようなサードパーティツールへの依存が発生する
- 個人開発の規模では同期リリースで十分なケースが多い

将来、skraft のユーザーから明確な需要が出た段階で、別 ADR (たとえば ADR 0011) で議論する余地を残す。

### 案 γ (リポジトリ分割) を否定しなかった理由

「skraft 管理下のリポジトリは 1 つのモノレポ」という設計を強制するわけではない。**スキル単位で更新頻度や監視主体が大きく異なる場合は、リポジトリを分けることを推奨する**。skraft はそれぞれのリポジトリに対して独立に動作する。

## 結果

### モノレポ前提の明示

skraft 管理下のリポジトリは複数スキルを含むモノレポを想定する。リポジトリ構造の例:

```
my-skills/
├── .skraft/                  ← skraft 管理ファイル
│   ├── config.toml
│   └── ledger.db
├── .gitignore
├── README.md
├── skill-a/
│   ├── SKILL.md
│   └── scripts/
├── skill-b/
│   └── SKILL.md
└── skill-c/
    ├── SKILL.md
    └── references/
```

### frontmatter スキーマの変更

skraft は SKILL.md の frontmatter を編集しない (ADR 0011 を参照)。本 ADR で確定した「`version` フィールドを書かない」方針は、より広い「**frontmatter に何も書かない**」方針の一部となった。

skraft が frontmatter を読む対象は標準フィールドのみ:

```yaml
---
name: my-skill           # Agent Skills 仕様の必須フィールド (skraft は読むだけ)
description: ...         # 同上
license: MIT             # 任意、skraft は読むだけ
allowed-tools: [...]     # 任意、skraft は読むだけ
---
```

skraft 固有の状態 (Claude.ai アップロード状態など) は SQLite ledger (ADR 0002) で管理する。バージョン情報は git tag から動的に取得する。

### `skraft status` の表示

バージョン情報は git から動的に取得して表示する。リポジトリ全体に 1 つの tag が切られているモデルなので、すべてのスキルで同じ tag を参照する:

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

git tag が存在しないリポジトリでは `Latest tag: untagged (5 commits)` のように表示する。

### `skraft pack` のファイル名

`pack` で生成する zip ファイル名にもバージョンを含める。リポジトリ全体の最新 tag を取得して命名:

```
dist/skill-a-v1.2.0.zip       ← 最新 tag が v1.2.0 で HEAD = tag
dist/skill-a-v1.2.0+3.zip     ← v1.2.0 + 3 commits ahead
dist/skill-a-untagged.zip     ← tag なし
```

### ADR 0002 (SQLite スキーマ) への影響

`upload_state.version` カラムは引き続き必要だが、ここに格納される値は **git tag の文字列 (例: `v1.2.0`) または HEAD の短縮 SHA (例: `abc123d`)** とする。frontmatter の値ではなく、`pack` 実行時に skraft が動的に取得した値を保存する。

```sql
CREATE TABLE upload_state (
    skill_name TEXT NOT NULL,
    target TEXT NOT NULL,
    version TEXT NOT NULL,       -- git tag または short SHA
    content_hash TEXT NOT NULL,
    uploaded_at TEXT NOT NULL,
    PRIMARY KEY (skill_name, target)
);
```

### `mark-uploaded` コマンドの引数

```bash
# デフォルト: 現在の git describe を自動取得
skraft mark-uploaded my-skill

# 明示指定 (緊急時用)
skraft mark-uploaded my-skill --as v1.2.0
```

デフォルトで現在の git tag (または short SHA) を採用することで、ユーザーがバージョンを書き間違えるリスクを排除する。

### ユーザー向けワークフロー

```bash
# リリースの流れ
git commit -am "fix: improve descriptions"
git tag v1.2.1                          # リポジトリ全体に 1 つの tag
git push origin main --tags

# skraft でアップロード状態を更新
skraft pack skill-a                     # dist/skill-a-v1.2.1.zip が生成
skraft pack skill-b                     # dist/skill-b-v1.2.1.zip が生成
# (Claude.ai に手動アップロード)
skraft mark-uploaded skill-a            # v1.2.1 が記録される
skraft mark-uploaded skill-b            # v1.2.1 が記録される
```

ユーザーは Git の標準ワークフローに従うだけで skraft が機能する。

### 解決すべき課題

#### tag のないリポジトリへの対応

新規作成直後で tag が存在しないリポジトリもある。この場合 `git describe --tags` は失敗するため、HEAD の短縮 SHA を使う。`untagged-abc123d` のように表示・命名する。

#### スキル単位の更新頻度差をどう扱うか

skill-a だけ頻繁に更新したいが、skill-b は触りたくない、というニーズが出る場合がある。同期リリースモデルでは「skill-b も v1.2.1 にバージョンアップする」が、内容は変わらないので実害はない。Claude.ai 側のアップロード状態管理 (`upload_state`) で「skill-b は v1.0.0 のまま、再アップロードは不要」と判断できる仕組みは別途必要。

#### 将来 prefix tag が必要になった場合

スキル間の独立性が極端に高まり、同期リリースが破綻するケースが出たら、別 ADR (たとえば ADR 0014) で prefix tag サポートを議論する。その際は `git describe --tags --match="<skill>/v*"` を使った per-skill バージョン取得を実装する。MVP ではスコープ外。

## 関連 ADR

- ADR 0002: SQLite を ledger のストレージとして採用 (upload_state.version の意味を変更)
- ADR 0011: skraft は SKILL.md frontmatter を編集しない (本 ADR の `version` 削除を frontmatter 全般に拡張)

## 参考

- [Agent Skills 仕様: https://agentskills.io/specification]
- [gh skill install: https://cli.github.com/manual/gh_skill_install]
- [gh skill publish (azukiazusa1 の解説): https://azukiazusa.dev/en/blog/gh-agent-skill-management/]
- [Git tag: https://git-scm.com/book/en/v2/Git-Basics-Tagging]
- [git describe: https://git-scm.com/docs/git-describe]
- [monorepo の prefix tag 戦略: https://medium.com/streamdal/monorepos-version-tag-and-release-strategy-ce26a3fd5a03]
