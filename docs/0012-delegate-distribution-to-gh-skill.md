# ADR 0012: 配布は `gh skill` に委譲する

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ
- **関連**: ADR 0009 (バージョニング戦略)

## 文脈

skraft はスキル作者向けのローカル開発支援ツールである。スキルライフサイクルは大きく分けて以下のフェーズで構成される:

1. **作成**: スキル本体を書く (skill-creator が担当)
2. **ローカル運用**: Claude Code で動作確認、Claude.ai に手動アップロード
3. **公開・配布**: 他のユーザーがインストールできるようにする
4. **インストール**: 利用者が自分の環境にスキルを取り込む
5. **更新検知**: 上流の変更をローカルに反映する

このうち、**3 と 4 と 5 (公開・配布、インストール、更新検知)** を担うツールとして `gh skill` が既に存在する:

- `gh skill publish` — リポジトリを Agent Skills 仕様に対して検証し、tag を切ってリリース
- `gh skill install` — 他のユーザーがスキルをエージェントホストに取り込む
- `gh skill update` — provenance metadata の git tree SHA を比較して上流の変更を検知
- `gh skill preview` — インストール前の安全確認

これは GitHub 公式が提供しており、Agent Skills エコシステムの**事実上の配布インフラ**になっている。Anthropic、OpenAI、HuggingFace、Vercel などの主要なスキル配布リポジトリも `gh skill` 経由でのインストールを前提にしている。

skraft が配布機能を自前で持つと、`gh skill` と機能が重複し、ユーザーは「skraft で配布するのか、gh skill で配布するのか」を選ぶ羽目になる。

## 検討した選択肢

| 候補 | 概要 | 評価 |
|---|---|---|
| **A. 配布は gh skill に完全委譲** | skraft は配布関連のコマンドを持たない | エコシステム整合、責務分離が明確 |
| B. skraft が独自に配布機能を持つ | `skraft publish` で GitHub Releases に push | gh skill と重複、自前で provenance 等を実装する必要 |
| C. skraft が gh skill のラッパーになる | `skraft publish` は内部で `gh skill publish` を呼ぶ | 薄い層を作るが、ユーザーは結局 gh skill の概念を学ぶ必要 |
| D. publisher 機能のみ skraft が持ち、installer は gh skill | `skraft publish` あり、`skraft install` なし | 中途半端、責務の線引きが恣意的 |

## 決定

**配布は `gh skill` に完全委譲する (案 A)。** skraft は配布関連のコマンドを一切持たない。

具体的には:

- `skraft publish` のようなコマンドは作らない
- `skraft install` のようなコマンドは作らない (skraft はそもそも installer 用途を想定しない)
- `gh skill publish` の前段としての validate も skraft は行わない (`gh skill publish --dry-run` が担当)

## 理由

### 1. 責務分離が明確になる

skraft の責務は「**publisher 側のローカル開発と環境間同期**」。配布層 (リポジトリの公開、他ユーザーへの提供) は別レイヤーであり、これを `gh skill` が担う。skraft はその前後の状態管理に集中する。

```
[作成] skill-creator
   ↓
[ローカル運用] skraft  ← skraft の責務はここ
   ↓
[配布] gh skill        ← skraft は触れない
   ↓
[インストール] gh skill (他ユーザー側)
```

### 2. エコシステム整合

`gh skill` は GitHub 公式で、Agent Skills 仕様に基づいて設計されている。Anthropic、OpenAI、HuggingFace、Vercel などの主要リポジトリがこのインフラに乗っている。skraft が独自の配布機構を持つと、エコシステムから外れた孤立した位置になる。

### 3. 重複実装の回避

`gh skill publish` は、Agent Skills 仕様に対する validate、tag protection、immutable releases、provenance metadata の生成など、**既に多くの機能を提供**している。skraft が同等の機能を自前で実装するのは無駄であり、メンテナンスコストも倍増する。

### 4. ユーザー体験の単純化

ユーザーが配布したいときは `gh skill publish` を実行する、それだけ。「skraft の publish」と「gh skill の publish」のどちらを使うか迷う必要がない。

### 5. 互換性の保証

skraft が gh skill の上流仕様 (provenance metadata の構造、tag の慣習など) と同期する責任を持たなくて済む。`gh skill` が将来仕様を変えても、skraft の責務に影響しない。

## 検討したが採用しなかった選択肢

### 案 B (独自配布) を採用しなかった理由

`skraft publish` で GitHub Releases に直接 push する独自実装は、技術的には可能だが以下の問題がある:

- `gh skill publish` が既に提供している validate、tag protection チェック、provenance 生成を自前で再実装する必要
- エコシステム標準 (Agent Skills 仕様) からずれるリスク
- 「skraft 経由で配布されたスキル」と「gh skill 経由で配布されたスキル」の互換性管理が必要になる

### 案 C (gh skill のラッパー) を採用しなかった理由

`skraft publish` が内部で `gh skill publish` を呼ぶ薄いラッパーにする案。一見便利だが、以下の問題がある:

- ユーザーは結局 `gh skill` の概念 (tag、scope、agent ホストなど) を理解する必要がある
- ラッパーが先回りしてオプションを限定すると、`gh skill` の機能拡張に追従できなくなる
- 「`skraft publish` と `gh skill publish` のどちらを使うべきか」という新しい混乱が生じる

ラッパーは抽象化に成功しないと、ユーザーに 2 つの API を覚えさせるだけになる。skraft の規模では成功する見込みが薄い。

### 案 D (publisher のみ skraft、installer は gh skill) を採用しなかった理由

線引きが恣意的で、ユーザーに「なぜ publisher は skraft で installer は gh skill なのか」を説明できない。責務分離としては「配布全体を gh skill に委譲」のほうが一貫する。

## 結果

### skraft が持たないコマンド

以下のコマンドは skraft には**存在しない**。これらが必要になった時はユーザーが直接 `gh skill` を使う:

- `skraft publish`
- `skraft install`
- `skraft search`
- `skraft preview`
- `skraft update` (上流の変更検知という意味での)

`skraft sync` は存在するが、これは「Claude Code・Claude.ai との状態同期」であり、上流リポジトリからの更新取得ではない。意味が異なる。

### ドキュメントでの誘導

README とヘルプメッセージで、「配布や他ユーザーへのインストールは `gh skill` を使う」と明示する。skraft は配布層を担わないことを早期に伝えることで、ユーザーの期待値を合わせる。

### `gh skill` がスキルに書き込む metadata の扱い

`gh skill install` は SKILL.md の frontmatter に provenance metadata を書き込む (ソースリポジトリ、ref、tree SHA)。これは ADR 0011 の「skraft は frontmatter を編集しない」方針と整合する。skraft は **`gh skill` が書いた metadata を読むことはあるが、書き換えない**。

これにより、skraft と gh skill は**書き込み領域が排他的**になり、相互運用が安全に保たれる。

### 利用者向け機能の不在

`gh skill install` でインストールしたスキルを利用する側のユーザーは、skraft のターゲットユーザーではない。skraft は publisher 側のツールであり、installer 側のサポート (インストール済みスキルの管理、更新通知など) は持たない。これが必要なユーザーは `gh skill` を直接使う。

### `gh skill publish` の前段でのチェック

`gh skill publish --dry-run --fix` で skill-creator や Agent Skills 仕様への適合をチェックできる。skraft はこの前段チェックを自前で持たない。ユーザーは `gh skill publish` 直前に `--dry-run` を実行する習慣を身につける。

### 将来 `gh skill` の仕様が変わったら?

`gh skill` の API や frontmatter スキーマが変わっても、skraft の動作には影響しない。skraft が依存しているのは「SKILL.md ファイルが存在すること」「Git でバージョン管理されていること」のみで、`gh skill` の内部実装には依存していない。

これは責務分離の利点で、両ツールが独立に進化できる。

## 関連 ADR

- ADR 0009: バージョニング戦略を git tag 中心に置く (`gh skill publish` のリリースモデルと整合)
- ADR 0011: skraft は SKILL.md frontmatter を編集しない (`gh skill` の書き込み領域と非衝突)

## 参考

- [gh skill 公式: https://cli.github.com/manual/gh_skill]
- [Agent Skills 仕様: https://agentskills.io/specification]
- [GitHub Changelog: gh skill 公開: https://github.blog/changelog/2026-04-16-manage-agent-skills-with-github-cli/]
