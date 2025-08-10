# ggsrun 改善提案 Issueリスト

このドキュメントは、`ggsrun` の潜在的な問題点を解決し、より堅牢で使いやすいツールにするための改善案をまとめたものです。各項目をタスクとして管理することを目的とします。

---

## 1. セットアップの複雑さ (完了)

### 現状の問題点
ユーザーが `ggsrun` を利用開始するまでのセットアッププロセス（GCPでのAPI有効化、認証情報作成、GASサーバーのデプロイなど）が非常に複雑で、手作業が多く、間違いが発生しやすい。

### 提案する解決策
対話形式でセットアップを支援する `ggsrun init` コマンドを新たに導入する。このコマンドがユーザーをガイドし、可能な限り多くの設定作業を自動化する。

### 具体的なタスクリスト
- [x] `urfave/cli` に `init` コマンドを追加する。
- [x] `ggsrun init` の基本的なコマンドフローを実装する。
- [x] `client_secret.json` の内容をターミナルから入力させ、設定を保存する機能を追加する。
- [x] Apps Script API を利用して、サーバーサイドスクリプト用のGASプロジェクトを自動で作成する機能を実装する。
- [x] 作成したGASプロジェクトに `server/server.gs` の内容をアップロードする機能を実装する。
- [x] ユーザーにGCPコンソールやGASエディタの特定ページへ直接誘導するためのURLを生成・表示する。
- [ ] `README.md` などのドキュメントを更新し、新しい `init` コマンドの利用方法を記載する。

---

## 2. サーバーサイドの `eval()` への依存 (完了)

### 現状の問題点
`server.gs` が `eval()` を使ってスクリプトを実行しているため、GAS側でエラーが発生した際に、CLIに返されるエラーメッセージが限定的でデバッグが困難。

### 提案する解決策
`eval()` の使用は維持しつつ、`server.gs` のエラーハンドリングを強化する。GAS側で発生したエラーの詳細（エラーメッセージ、スタックトレース、行番号など）をJSON形式でCLIに返し、デバッグを容易にする。

### 具体的なタスクリスト
- [x] `server.gs` の `try...catch` ブロックを修正し、`err` オブジェクトから詳細な情報（`name`, `message`, `stack`）を抽出するロジックを追加する。
- [x] 抽出したエラー情報を格納するためのJSON構造を定義する。
- [x] Goクライアント (`sender.go`) 側で、APIレスポンスがGASエラーであるかを判定するロジックを改善する。
- [x] 詳細なエラー情報JSONをパースし、ユーザーが見やすい形式に整形して表示する機能を実装する。

---

## 3. エラーハンドリング (`os.Exit(1)`) (一部完了)

### 現状の問題点
Goのコード内でエラーが発生すると `os.Exit(1)` によりプログラムが即時終了するため、柔軟なエラー処理やリソースのクリーンアップができない。

### 提案する解決策
Goの標準的なエラーハンドリングパターンにリファクタリングする。各関数が `error` 型を返し、呼び出し元がエラーをハンドリングできるようにする。

**対応状況:** `exe1` コマンドのフローを中心にリファクタリングを実施。他のコマンドフローは未対応。

### 具体的なタスクリスト
- [x] `handler.go` 内のメソッドチェーンを解消し、各関数が `error` を返すようにシグネチャを変更する。 (`exeAPIWithout` のみ完了)
- [x] `ggsrunIni`, `goauth` などのコア関数から `os.Exit(1)` の呼び出しを削除し、代わりに `return err` を使用する。
- [x] `fmt.Errorf("...: %w", err)` を使用して、エラーにコンテキスト情報を付与する。
- [x] `urfave/cli` のActionハンドラ（`exeAPIWithout`など）で最終的なエラーを受け取り、ユーザーにメッセージを表示して終了するように修正する。 (`exeAPIWithout` のみ完了)

---

## 4. 巨大な構造体による状態管理 (一部完了)

### 現状の問題点
`AuthContainer` や `ExecutionContainer` といった巨大な構造体が全ての状態を保持し、メソッドチェーンで引き回されている。これによりコンポーネントが密結合になり、テストやメンテナンスが困難になっている。

### 提案する解決策
依存性の注入（Dependency Injection）の設計思想に基づき、リファクタリングを行う。各関数は巨大な構造体に依存するのではなく、必要なデータだけを引数として受け取るようにする。

**対応状況:** `ggsrunIni` 関数を `AuthContainer` から分離し、独立した関数 `doGgsrunIni` としてリファクタリングする最初のステップを完了。`AuthContainer` と `ExecutionContainer` のコンストラクタを独立させ、`exe1Function` とそのヘルパー関数を独立させた。

### 具体的なタスクリスト
- [x] まずは1つの関数（例: `ggsrunIni`）を対象に、リファクタリングのプルーフ・オブ・コンセプトを実装する。
    - [x] `ggsrunIni` の関数シグネチャを、コンテナ構造体ではなく、必要な値を引数で受け取るように変更する。
    - [x] `ggsrunIni` が生成した値を戻り値で返すように変更する。
    - [x] `handler.go` の呼び出し箇所を修正し、新しいシグネチャに対応させる。
- [x] 他の関数についても、同様のリファクタリングを段階的に進める。
    - [x] `defAuthContainer` を `newAuthContainer` にリファクタリングし、独立した関数にする。
    - [x] `defExecutionContainer` を `newExecutionContainer` にリファクタリングし、独立した関数にする。
    - [x] `exe1Function` を `doExe1Function` にリファクタリングし、独立した関数にする。
    - [x] `projectBackup` を `doProjectBackup` にリファクタリングし、独立した関数にする。
    - [x] `projectUpdateIni` を `doProjectUpdateIni` にリファクタリングし、独立した関数にする。
    - [x] `projectUpdate2` を `doProjectUpdate2` にリファクタリングし、独立した関数にする。

**残りのリファクタリングタスク:**
- [x] `exeAPIWithout` コマンドフローの残りの関数をリファクタリングする。
    - [x] `executionAPIwithoutServer` を独立した関数 `doExecutionAPIwithoutServer` にする。
    - [x] `esenderForExe1` を独立した関数 `doEsenderForExe1` にする。
    - [x] `dispResult` を独立した関数 `doDispResult` にする。
    - [x] `executionError` を独立した関数 `doExecutionError` にする。
    - [x] `chkAtokenForExecution` を独立した関数 `doChkAtokenForExecution` にする。
- [ ] `exeAPIWithout` 以外のコマンドフローをリファクタリングする。
    - [x] `defExecutionContainerWebApps` を独立した関数にする。
    - [x] `defDownloadContainer` を独立した関数にする。
    - [x] `defUploadContainer` を独立した関数にする。
    - [ ] `defPermissionsContainer` を独立した関数にする。
    - [ ] `dispUpdateProjectContainer` を独立した関数にする。
    - [ ] `defDownloadByScriptContainer` を独立した関数にする。
    - [ ] `defUpdateProjectContainer` を独立した関数にする。
    - [ ] `convExecutionContainerToFileInf` を独立した関数にする。
    - [x] `exe2Function` を独立した関数にする。
    - [x] `webAppsWith` (ハンドラ) をリファクタリングする。
    - [ ] `downloadFiles` (ハンドラ) をリファクタリングする。
    - [ ] `uploadFiles` (ハンドラ) をリファクタリングする。
    - [ ] `updateProject` (ハンドラ) をリファクタリングする。
    - [ ] `revisionFiles` (ハンドラ) をリファクタリングする。
    - [ ] `showFileList` (ハンドラ) をリファクタリングする。
    - [ ] `searchFilesByQueryAndRegex` (ハンドラ) をリファクタリングする。
    - [ ] `managePermissions` (ハンドラ) をリファクタリングする。
    - [ ] `getDriveInformation` (ハンドラ) をリファクタリングする。
    - [ ] `reAuth` (ハンドラ) をリファクタリングする。
- [ ] 長期的なゴールとして、全ての関数がリファクタリングされた後、不要になったコンテナ構造体を削除する。