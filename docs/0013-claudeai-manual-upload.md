# ADR 0013: Claude.ai は手動アップロード前提とする

- **ステータス**: Accepted
- **日付**: 2026-05-03
- **決定者**: メンテナ
- **関連**: ADR 0002 (SQLite ledger), ADR 0009 (バージョニング戦略), ADR 0012 (配布は gh skill に委譲)

## 文脈

skraft は Claude のスキルを以下の 3 つの環境間で同期させる:

- **Git リポジトリ** (source of truth)
- **Claude Code** (ローカルファイルシステム)
- **Claude.ai** (consumer Web 製品)

このうち Claude.ai は他の 2 つと性質が大きく異なる:

- **API が存在しない**: Claude.ai (consumer Web、claude.ai ドメイン) には custom skill のアップロードを行うプログラマティックな手段がない
- **手段はブラウザ UI のみ**: ユーザーは Settings → Capabilities → Skills (もしくは Customize → Skills) で zip ファイルを手動アップロードする必要がある
- **ユーザーごとに個別**: Pro/Max/Team/Enterprise プランで個別管理され、組織横断的なプログラマティック制御もない (一部 Team/Enterprise で管理者プロビジョニング機能はあるが、個人利用とは別経路)

GitHub の `anthropics/claude-code#25771` でも、「Claude.ai 側のプログラマティックなデプロイ」は feature request として上がっており、まだ実装されていない。

注: Claude API (`api.anthropic.com`) には Skills API (`POST /v1/skills`) が存在し、プログラマティックなアップロードが可能。しかし Claude.ai と Claude API は別製品であり、別のスコープで動作する。本 ADR は **Claude.ai (consumer Web) の話** に限定する。

skraft は publisher 側のローカル開発支援ツールであり (ADR 0012)、Claude.ai は重要な実行環境の一つ。これを sync 対象から外すと、skraft の価値が大きく損なわれる。

## 検討した選択肢

| 候補 | 概要 | 評価 |
|---|---|---|
| **A. 手動アップロード前提** | skraft は zip 生成と状態記録のみ、アップロードはユーザー作業 | 構造的制約に従う、堅実 |
| B. ブラウザ自動操作 | Playwright 等で claude.ai の UI を自動操作 | 脆弱、TOS リスク、メンテナンス困難 |
| C. Claude.ai サポートを諦める | Claude Code のみ対応 | skraft の価値が半減 |
| D. Claude.ai 側 API の追加を待つ | Anthropic が API を提供するまで対応保留 | 待つ間 skraft が機能しない |

## 決定

**手動アップロード前提とする (案 A)。** skraft は以下の機能で Claude.ai サポートを実現する:

1. **`skraft pack`**: アップロード用の zip を生成 (Claude.ai が要求する形式に準拠)
2. **ユーザーがブラウザでアップロード**: skraft の責務外
3. **`skraft mark-uploaded`**: アップロード完了をユーザーが skraft に伝える
4. **`skraft sync --check`**: ledger に記録された状態と Git 上の最新版を比較して、再アップロードが必要なスキルを表示

skraft は Claude.ai に対しては**観測点 (mark-uploaded) を提供する**ことで状態管理に参加する。直接の操作はしない。

## 理由

### 1. 構造的制約に従う

Claude.ai に API がない以上、skraft が直接アップロードする手段は存在しない。これは skraft の設計判断ではなく、Claude.ai 側の仕様に起因する制約である。**ない API を呼ぶ方法を考えるより、ある手段で価値を最大化する**ほうが建設的。

### 2. ブラウザ自動操作は脆弱

Playwright や Puppeteer で claude.ai の UI を自動操作する案は技術的には可能だが、以下の問題がある:

- **UI 変更で壊れる**: Claude.ai の UI レイアウトや DOM 構造の変更で動作不能になる。Anthropic は予告なく UI を変更する可能性がある
- **認証の難しさ**: claude.ai のログインは Google/Apple OAuth や Magic Link などで、自動化が複雑かつセキュリティリスクが高い
- **TOS のグレーゾーン**: 利用規約上、自動化が許容されるか不明瞭。違反した場合のアカウント停止リスクがある
- **メンテナンス負荷**: UI 変更のたびに skraft 側で対応が必要

「動くこともあるが、いつ壊れるか分からないツール」をユーザーに提供するのは無責任。

### 3. Claude.ai サポートを諦めると skraft の価値が大きく減る

skraft の主要価値は「Git → Claude Code → Claude.ai の同期」である。Claude.ai を sync 対象から外すと、Claude Code 単体の link 管理ツールに過ぎなくなり、`gh skill install --from-local` 等で代替可能になってしまう。

Claude.ai の手動アップロードを支援する機能こそが、skraft が他ツールと差別化できる領域。

### 4. 「半自動」は構造を理解させる UX としても優れる

ユーザーが「ブラウザでアップロードする」というステップを意識的に踏むことで、Claude.ai 側で何が起きているかを理解できる。完全自動化よりも、**境界が明示される**ほうが学習コストが下がる場合がある。

将来 Claude.ai に API が追加されたら、`skraft push` のようなコマンドを追加して半自動から完全自動に進化できる。

### 5. 実装が単純

zip 生成 + 状態記録だけで構成されるので、実装が単純。Playwright 等の重い依存を skraft が抱える必要がない。

## 検討したが採用しなかった選択肢

### 案 B (ブラウザ自動操作) を採用しなかった理由

技術的には可能でも、**メンテナンス・セキュリティ・利用規約**の三重の問題がある。個人開発のツールがこれを抱えるのは持続不可能。Anthropic 公式の API 追加を待つほうが筋が良い。

### 案 C (Claude.ai サポートを諦める) を採用しなかった理由

Claude.ai は Pro/Max/Team プランの主要ユーザーが日常的に使う環境。これをサポートしないと skraft の価値が大きく減る。`skraft pack` + `mark-uploaded` の組み合わせは構造的制約の中で最大の価値を提供する。

### 案 D (API 追加を待つ) を採用しなかった理由

Anthropic が Claude.ai 側の skill アップロード API を追加するかどうか、いつ追加するかは不明。待っている間に skraft が機能しないのは本末転倒。**現時点で提供可能な機能を提供し、API が追加されたらアップグレードする**ほうが現実的。

## 結果

### `skraft pack` の仕様

```bash
skraft pack [skill-name]
```

- 引数なし: リポジトリ内の全スキルの zip を生成
- 引数あり: 指定スキルのみの zip を生成
- 出力先: `dist/<skill-name>-<version>.zip` (`<version>` は ADR 0009 の git tag ベース)
- zip の内容: スキルディレクトリ全体 (`SKILL.md`、`scripts/`、`references/`、`assets/` など)
- 除外: `.git`、`.DS_Store`、`node_modules`、その他の非配布ファイル

### `skraft mark-uploaded` の仕様

```bash
skraft mark-uploaded <skill-name>           # 現在の git tag を自動取得
skraft mark-uploaded <skill-name> --as v1.2.0  # 明示指定 (緊急時用)
```

- ledger の `upload_state` テーブルに以下を記録:
  - `skill_name`: スキル名
  - `target`: `claudeai`
  - `version`: git tag または short SHA
  - `content_hash`: 直近の `pack` 結果の zip の SHA256
  - `uploaded_at`: 実行時刻 (ISO 8601)

### `skraft sync --check` の挙動

Claude.ai に対しては以下の比較を行う:

1. 現在の git tag (リポジトリの状態)
2. ledger に記録された `claudeai` ターゲットの最新 `version`

これらが一致しない場合、「再アップロードが必要」と表示する:

```
$ skraft sync --check
skill-a  ✓ Claude.ai: v1.2.0 (current)
skill-b  ⚠ Claude.ai: v1.1.0 (current is v1.2.0, re-upload needed)
         dist/skill-b-v1.2.0.zip is ready
skill-c  ⚠ Claude.ai: never uploaded
         dist/skill-c-v1.2.0.zip is ready
```

### `skraft sync --fix` の挙動

Claude.ai 側のずれは**自動修復しない** (アップロードが手動なため)。代わりに、必要な zip パスとアップロード手順を明示する:

```
$ skraft sync --fix
Claude Code: 2 skills re-linked.

Claude.ai: manual action required.
  skill-b: Upload dist/skill-b-v1.2.0.zip via claude.ai Settings → Capabilities → Skills
  skill-c: Upload dist/skill-c-v1.2.0.zip via claude.ai Settings → Capabilities → Skills

After uploading, run:
  skraft mark-uploaded skill-b
  skraft mark-uploaded skill-c
```

### content_hash を保存する理由

`upload_state.content_hash` は、ledger の version と実際にアップロードされた zip の内容を紐付ける。同じバージョン番号でも、`pack` のロジック変更や除外ファイルの追加で zip 内容が変わる可能性がある。content_hash を残すことで、「version は v1.2.0 だが、最新の pack 結果と異なる」という検知が可能になる。

ただし MVP では content_hash の比較ロジックは実装せず、記録のみ行う。将来必要に応じて活用する。

### Claude.ai 側の状態は信頼するしかない

skraft は Claude.ai 側の実際の状態を観測できない (API がないため)。`mark-uploaded` でユーザーが「アップロードした」と申告した内容を信頼するしかない。

これはモデル上の制約で、skraft が解決できる問題ではない。ユーザーが嘘の `mark-uploaded` を実行することで状態が嘘になるが、それはユーザー自身を欺くことになるので、実用上問題にならない。

### 将来 Claude.ai に API が追加された場合

Anthropic が将来 Claude.ai consumer Web 向けの skill アップロード API を提供したら、本 ADR を改訂して `skraft push` のようなコマンドを追加できる。その場合も既存の `pack` + `mark-uploaded` の組み合わせは互換性のため残す可能性がある (オフライン環境やネットワーク制限下での運用)。

## 解決すべき課題

### Claude.ai の UI が変わったら表示文言を更新する必要

`sync --fix` の出力に「Settings → Capabilities → Skills」と書いているが、Claude.ai の UI 変更でメニュー名が変わる可能性がある (実際 Pro 用は Customize → Skills という記述もある)。skraft は UI 文言をハードコードしない。

代わりに「Claude.ai の skill 設定画面」のようなニュートラルな表現を使う。詳細な手順はリリースノートか README で別途伝える。

### Team/Enterprise プランの管理者プロビジョニング

Team/Enterprise プランでは管理者が組織全体に skill をプロビジョニングできる。これは個人利用とは別経路で、skraft の MVP ではサポートしない。`mark-uploaded` は個人アカウントへのアップロードを前提とする。

組織プロビジョニングが必要なユーザーは、Anthropic の管理者向け機能を直接使う。

## 関連 ADR

- ADR 0002: SQLite を ledger のストレージとして採用 (`upload_state` テーブルで状態を保持)
- ADR 0009: バージョニング戦略を git tag 中心に置く (`upload_state.version` の値の決定方法)
- ADR 0012: 配布は gh skill に委譲する (publisher 用ツールという位置づけの根拠)

## 参考

- [Use Skills in Claude (Help Center): https://support.claude.com/en/articles/12512180-use-skills-in-claude]
- [Feature request: Programmatic skill deployment via CLI or API: https://github.com/anthropics/claude-code/issues/25771]
- [Claude API Skills (別製品): https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview]
