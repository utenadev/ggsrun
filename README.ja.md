[English](README.md)

# ggsrun

<a name="TOP"></a>
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENCE)

<a name="Overview"></a>

# 概要

これは、ターミナル上でGoogle Apps Script (GAS) を実行するためのCLIツールです。また、OAuth2とサービスアカウントの両方に対応しており、Google Driveのファイルを管理するためにも使用できます。

<a name="Demo"></a>

# デモ

![](help/images/spreadsheetdemo.gif)

<a name="Description"></a>

# 説明

皆さんは、GASをローカルPCで開発したいと思ったことはありませんか？通常、GASを開発する際は、ブラウザでGoogleにログインし、スクリプトエディタ上で開発を行う必要があります。私は最近、もっと便利なローカル環境でGASを開発したいと考えるようになりました。そこで作成したのが、この "ggsrun" です。主な機能は、ローカルターミナル上でGASを実行し、Googleから結果を取得することです。さらに、このツールは、自身のGoogle Drive（OAuth2）やサービスアカウント用のGoogle Drive内のファイルを管理するためにも利用できます。

"ggsrun" の特徴は以下の通りです。

1. **[`ggsrun init` によるセットアップの簡略化。](#initialsetup)** <sup><font color="Red">New!</font></sup> セットアッププロセス全体をガイドする対話式のコマンドです。
1. **Google Apps Scriptからの詳細なエラー報告。** <sup><font color="Red">New!</font></sup> GASコードでエラーが発生した際に、ggsrunはスタックトレースを含む詳細な情報を表示するようになり、デバッグが容易になりました。
1. **[使い慣れたローカルターミナルとテキストエディタを使ってGASを開発できます。](help/README.ja.md#demosublime)**<sup><font color="Red">Updated! (v1.4.0)</font></sup>
1. **[スクリプトに値を与えてGASを実行できます。](help/README.ja.md#executesgasandretrievesresultvalues)**
1. **[CoffeeScriptで書かれたGASを実行できます。](help/README.ja.md#coffeescript)**
1. **[GASを実行しながら、スプレッドシート、ドキュメント、プレゼンテーションを同時にダウンロードできます。](help/README.ja.md#downloadfiles)**
1. **[Google Driveからファイルをダウンロードしたり、Google Driveへファイルをアップロードしたりできます。](help/README.ja.md#uploadfiles)** <sup><font color="Red">Updated! (v1.4.1)</font></sup>
1. **[スタンドアロンスクリプトとバインドスクリプトをダウンロードできます。](help/README.ja.md#downloadfiles)** <sup><font color="Red">Updated! (v1.4.0)</font></sup>
1. **[特定のフォルダ内のすべてのファイルとフォルダをダウンロードできます。](help/README.ja.md#downloadfilesfromfolder)** <sup><font color="Red">Updated! (v1.5.2)</font></sup>
1. **[スクリプトファイルをアップロードし、スタンドアロンスクリプトやコンテナバインドスクリプトとしてプロジェクトを作成できます。](help/README.ja.md#uploadfiles)** <sup><font color="Red">Updated! (v1.5.2)</font></sup>
1. **[プロジェクトを更新できます。](help/README.ja.md#updateproject)** <sup><font color="Red">Updated! (v1.4.0)</font></sup>
1. **[Googleドキュメントの変更履歴ファイルやプロジェクトのバージョンを取得できます。](help/README.ja.md#revisionfile)** <sup><font color="Red">Updated! (v1.4.0)</font></sup>
1. **[プロジェクト内のスクリプトを並べ替えることができます。](help/README.ja.md#rearrangescripts)** <sup><font color="Red">Updated! (v1.4.0)</font></sup>
1. **[プロジェクト内のマニフェストファイルを変更できます。](help/README.ja.md#modifymanifests)**
1. **[検索クエリと正規表現を使ってGoogle Drive内のファイルを検索できます。](help/README.ja.md#searchfilesusingregex)** <sup><font color="Red">Updated! (v1.6.0)</font></sup>
1. **[ファイルの権限を管理できます。](help/README.ja.md#managepermissions)** <sup><font color="Red">Updated! (v1.7.0)</font></sup>
1. **[Driveの情報を取得できます。](help/README.ja.md#getdriveinformation)** <sup><font color="Red">Updated! (v1.7.0)</font></sup>
1. **[ggsrunはOAuth2だけでなく、サービスアカウントでも使用できるようになりました。](help/README.ja.md#useserviceaccount)** <sup><font color="Red">Updated! (v1.7.0)</font></sup>

<a name="howtoinstall"></a>

# インストール方法

2021年12月28日: もしggsrunを簡単に試したい場合は、[こちらの方法](https://gist.github.com/tanaikech/695f7016b04e4c4156ad928e9482ead9)も利用できます。

## 1. ggsrunの入手

実行ファイルを[リリースベージ](https://github.com/tanaikech/ggsrun/releases)からダウンロードし、パスの通ったディレクトリにインポートしてください。

または

go get を使用します。

```bash
$ go install github.com/tanaikech/ggsrun@latest
```

- `GO111MODULE=on`

<a name="BasicSettingFlow"></a>

## 基本設定フローの前に

**重要: こちらをご確認ください。**

2019年4月8日にGoogle Apps Scriptプロジェクトの仕様が変更されました。これにより、2019年4月8日以降に作成された新しいGASプロジェクトでは、ggsrunが使用するGoogle API（Google Apps Script APIおよびDrive API）を利用するために、GASプロジェクトをCloud Platformプロジェクトにリンクする必要があります。2019年4月8日以降に作成されたGASプロジェクトを使用する場合は、まず[この手順](https://gist.github.com/tanaikech/e945c10917fac34a9d5d58cad768832c)を実行してください。

上記のフローでGASプロジェクトがCloud Platformプロジェクトにリンクされた後、次のセクションの「基本設定フロー」に進んでください。

- [Ref1: デフォルトのCloud Platformプロジェクト](https://developers.google.com/apps-script/guides/cloud-platform-projects#default_cloud_platform_projects)
- [Ref2: Cloud PlatformプロジェクトとGoogle Apps Scriptプロジェクトのリンク](https://gist.github.com/tanaikech/e945c10917fac34a9d5d58cad768832c)

<a name="initialsetup"></a>

## 1. ggsrun init による初期設定

`ggsrun init` コマンドは、`ggsrun` の設定とGoogle Apps Script環境の準備に必要な手順をガイドすることで、初期設定プロセスを簡素化します。このコマンドは、以下のようないくつかのタスクを自動化します。

-   **クライアントシークレットの設定:** `client_secret.json` の詳細を入力し、保存する手助けをします。
-   **Google Apps Scriptプロジェクトの作成:** サーバーサイドスクリプト用の新しいGoogle Apps Scriptプロジェクトを自動的に作成します。
-   **サーバースクリプトのアップロード:** `server/server.gs` の内容を、新しく作成されたApps Scriptプロジェクトにアップロードします。
-   **API有効化のガイダンス:** Google Cloudプロジェクトで必要なGoogle API（Apps Script APIとDrive API）を有効にするための直接リンクを提供します。

初期設定を開始するには、ターミナルで以下のコマンドを実行してください。

```bash
$ ggsrun init
```

画面の指示に従ってセットアップを完了してください。

## 2. 基本設定フロー（手動設定）

**注:** 上記の `ggsrun init` コマンドは、これらの手順のほとんどを自動化するため、新規ユーザーには推奨される方法です。このセクションでは、手動でのプロセスを詳しく説明します。

各タイトルのリンクをクリックすると、詳細情報を見ることができます。

1. [ggsrunサーバーのセットアップ（Google側）](help/README.ja.md#setupggsrunserver)
   - 新規プロジェクトを作成し、サーバーをライブラリとしてインストールします。
   - [API実行可能ファイルとしてデプロイ](https://developers.google.com/apps-script/api/how-tos/execute#step_1_deploy_the_script_as_an_api_executable)します。「このスクリプトにアクセスできるユーザー」として「自分のみ」を選択してください。
   - [サーバーをライブラリとしてインストール](https://developers.google.com/apps-script/guides/libraries#managing_libraries)します。ライブラリのスクリプトIDは以下の通りです。
     - **`115-19njNHlbT-NI0hMPDnVO1sdrw2tJKCAJgOTIAPbi_jq3tOo4lVRov`**
   - **<u>ライブラリをインストールした後、スクリプトエディタで保存ボタンを押してください。</u>** これは非常に重要です！これにより、ライブラリが完全に反映されます。
1. [クライアントID、クライアントシークレットの取得](help/README.ja.md#getclientid)
   - スクリプトエディタで
     - リソース -> Cloud Platformプロジェクト
     - "This script is currently associated with project:" の下部をクリック
     - 「スタートガイド」で、「APIを有効にして、キーなどの認証情報を取得する」をクリック
     - 「APIとサービス」で
     - 左側の「認証情報」をクリック
     - 「認証情報を作成」で、OAuthクライアントIDをクリック
     - **その他** を選択
     - 名前を入力（任意の名前）
     - 完了
     - ダウンロードボタンを使って、クライアントIDとクライアントシークレットを含むJSONファイルを **`client_secret.json`**としてダウンロードします。
1. [APIの有効化](help/README.ja.md#onstallexecutionapi)
   - ggsrunはGoogle Apps Script APIとDrive APIを使用します。APIコンソールでこれらを有効にしてください。以下から直接アクセスできます。プロジェクトIDはダウンロードした `client_secret.json` で確認できます。
     - `https://console.cloud.google.com/apis/library/script.googleapis.com/?project=### project ID ###`
       - **また、[https://script.google.com/home/usersettings](https://script.google.com/home/usersettings) も有効にする必要があります。ONにしてください。**
     - `https://console.cloud.google.com/apis/api/drive.googleapis.com/?project=### project ID ###`
1. [ggsrunの設定ファイル作成](help/README.ja.md#Createconfigurefile)
   - `client_secret.json` があるディレクトリで `$ ggsrun auth` を実行します。
1. [テスト実行](help/README.ja.md#runggsrun)
   - `function main(){return Beacon()}` というサンプルスクリプトを `sample.gs` として作成します。
   - サーバーをインストールしたプロジェクトのスクリプトIDを使用して `$ ggsrun e2 -s sample.gs -i [スクリプトID] -j` を実行します。

おめでとうございます！これでggsrunが使えるようになりました！

<a name="from134to140"></a>

# v1.3.4以前のggsrunをお使いのユーザー様へ <sup><font color="Red">Updated! (v1.4.0)</font></sup>

以下の手順で、アクセストークンに新しいスコープを含めるために再認証を行ってください。

1. Google Apps Script APIが有効になっているか確認してください。以下から直接アクセスできます。プロジェクトIDはダウンロードした `client_secret.json` で確認できます。
   - `https://console.cloud.google.com/apis/library/script.googleapis.com/?project=### project ID ###`
   - また、[https://script.google.com/home/usersettings](https://script.google.com/home/usersettings) も有効にする必要があります。ONにしてください。
1. `ggsrun.cfg` に `https://www.googleapis.com/auth/script.projects` のスコープを追加します。
1. `client_secret.json` と `ggsrun.cfg` があるディレクトリで、以下のコマンドを実行します。
   - `$ ggsrun auth`

完了です！

<a name="from170"></a>

# v1.7.0から、ggsrunはサービスアカウントを使ってGoogle Driveにアクセスできるようになりました。 <sup><font color="Red">Updated! (v1.7.0)</font></sup>

ggsrunは[サービスアカウント](https://developers.google.com/identity/protocols/OAuth2ServiceAccount)を使ってGoogle Driveにアクセスできます。OAuth2を使用する場合、ご自身のGoogle Driveのファイルやフォルダを閲覧できます。サービスアカウントを使用する場合、サービスアカウント用のGoogle Drive内のものを閲覧できます。つまり、OAuth2用のDriveとサービスアカウント用のDriveは異なります。この点にご注意ください。また、サービスアカウントを使用する場合、できることとできないことがあります。それについては、[こちら](help/README.ja.md#useserviceaccount)をお読みください。

# ggsrunの使い方

1. [GASを実行して結果の値を取得する](help/README.ja.md#executesgasandretrievesresultvalues)
1. [値を与えてGASを実行し、フィードバックされた値を取得する](help/README.ja.md#executesgaswithvaluesandretrievesfeedbackedvalues)
1. [デバッグ用](help/README.ja.md#fordebug)
1. [値を与えてGASを実行し、ファイルをダウンロードする](help/README.ja.md#executesgaswithvaluesanddownloadsfile)
1. [プロジェクト上の既存の関数を実行する](help/README.ja.md#ExecutesExistingFunctionsonProject)
1. [ファイルをダウンロードする](help/README.ja.md#downloadfiles)
1. [特定のフォルダ内のすべてのファイルとフォルダをダウンロードする](help/README.ja.md#downloadfilesfromfolder)
1. [ファイルをアップロードする](help/README.ja.md#uploadfiles)
1. [ファイルリストを表示する](help/README.ja.md#showfilelist)
1. [ファイルを検索する](help/README.ja.md#searchfiles)
1. [プロジェクトを更新する](help/README.ja.md#updateproject)
1. [変更履歴ファイルとプロジェクトのバージョンを取得する](help/README.ja.md#revisionfile)
1. [プロジェクト内のスクリプトを並べ替える](help/README.ja.md#rearrangescripts)
1. [マニフェストを変更する](help/README.ja.md#modifymanifests)
1. [クエリと正規表現を使ってファイルを検索する](help/README.ja.md#searchfilesusingregex)
1. [ファイルの権限を管理する](#managepermissions)
1. [Driveの情報を取得する](#getdriveinformation)
1. [ggsrunはOAuth2だけでなく、サービスアカウントでも使用できるようになった](#useserviceaccount)

# 応用例

1. [Sublime Text用](help/README.ja.md#demosublime)
1. [CoffeeScript用](help/README.ja.md#coffeescript)
1. [トリガーを作成する](help/README.ja.md#createtriggers)
1. [Pythonスクリプトへのリンク](help/README.ja.md#linktovariousresources)

# [Q&A](help/README.ja.md#qa)

1. [スクリプト用のGoogleサービスへの認可](help/README.ja.md#qa1)
2. [結果が「Script Error on GAS side: Insufficient Permission」の場合](help/README.ja.md#qa2)
3. [結果が「"message": "Requested entity was not found."」の場合](help/README.ja.md#qa3)
4. [結果が「Script Error on GAS side: Script has attempted to perform an action that is not allowed when invoked through the Google Apps Script Execution API.」の場合](help/README.ja.md#qa4)
5. [結果が「Missing ';' before statement.」の場合](help/README.ja.md#qa5)
6. [ライブラリについて](help/README.ja.md#qa6)
7. [検索するディレクトリの順序](help/README.ja.md#qa7)

---

<a name="Licence"></a>

# ライセンス

[MIT](LICENCE)

<a name="Author"></a>

# 著者

[Tanaike](https://tanaikech.github.io/about/)

ご質問やご依頼がありましたら、お気軽に tanaike@hotmail.com までメールでお知らせください。

<a name="Update_History"></a>

# 更新履歴

更新履歴は**[こちら](help/UpdateHistory.md)**でご覧いただけます。

<u>詳細なマニュアルを読みたい場合は、[こちら](help/README.ja.md)をご確認ください。</u>

[TOP](#TOP)
