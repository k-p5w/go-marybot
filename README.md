# go-marybot



## TwitchBOT

- チャット欄に常駐するBOTを作る

### 機能

- はじめてのひとに挨拶
- 日本語へ翻訳
- ビッツに感謝を示す
- メッセージ数の記録

## 動かし方

### 認証

https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/

https://id.twitch.tv/oauth2/authorize?response_type=token&client_id=i076k2hmozpb75y3i7mod3kdm2vtb5&redirect_uri=http://localhost:3000&scope=channel%3Amanage%3Apolls+channel%3Aread%3Apolls&state=c3ab8aa609ea11e793ae92361f002671

- スコープ的にコレ（↓）じゃないとダメそう。

https://id.twitch.tv/oauth2/authorize?client_id=ko0iqmkptezsp3z0ij87r6zwgtdhxz&redirect_uri=http%3A%2F%2Flocalhost%3A3000&response_type=code&scope=chat%3Aread+chat%3Aedit

### 起動

> go run .

<!--  Hi x-san, good morning! -->

## 名前の由来

「Mary」＋「bot」：
　人間らしい名前「Mary（メアリー）」と、人工的存在を示す「bot（ロボット）」を組み合わせて、「親しみやすくて、でも機械的な存在」というキャラづけ(かもしれません。)


## メモ

ID,カテゴリ名,配信者数,全体の視聴者数,TOP10の視聴者数,TOP10以降の視聴者数
509658,雑談,96,135299,73138,62161,2025/06/26 18:14

Category ID:10229[FINAL FANTASY XI ONLINE] 
Category ID:24241[FINAL FANTASY XIV ONLINE] 
509658:マイクラ
10229:雑談
