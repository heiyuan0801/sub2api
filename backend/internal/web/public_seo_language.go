package web

type seoText struct {
	title       string
	description string
}

type seoLanguage struct {
	code      string
	tag       string
	openGraph string
	home      seoText
	market    seoText
	docs      seoText
}

// Keep these labels aligned with frontend/src/i18n locale metadata.
var seoLanguages = []seoLanguage{
	{
		code: "en", tag: "en", openGraph: "en_US",
		home:   seoText{"AI Model API Gateway", "Access AI models through a unified API and browse model pricing and integration guides."},
		market: seoText{"Model Market", "Compare AI models and see input, output and cache prices by group."},
		docs:   seoText{"API Documentation", "Learn how to connect to the AI API, configure clients and manage API keys."},
	},
	{
		code: "zh", tag: "zh-Hans", openGraph: "zh_CN",
		home:   seoText{"AI 模型 API 网关", "通过统一 API 接入 AI 模型，查看模型定价和接入指南。"},
		market: seoText{"模型市场", "按分组比较 AI 模型的输入、输出和缓存价格。"},
		docs:   seoText{"API 文档", "查看 AI API 接入、客户端配置和 API 密钥管理指南。"},
	},
	{
		code: "zh-TW", tag: "zh-Hant", openGraph: "zh_TW",
		home:   seoText{"AI 模型 API 閘道", "透過統一 API 串接 AI 模型，查看模型價格與串接指南。"},
		market: seoText{"模型市場", "依群組比較 AI 模型的輸入、輸出與快取價格。"},
		docs:   seoText{"API 文件", "查看 AI API 串接、用戶端設定與 API 金鑰管理指南。"},
	},
	{
		code: "ja", tag: "ja", openGraph: "ja_JP",
		home:   seoText{"AIモデル APIゲートウェイ", "統一APIでAIモデルに接続し、モデルの料金と導入ガイドを確認できます。"},
		market: seoText{"モデルマーケット", "AIモデルの入力・出力・キャッシュ料金をグループ別に比較できます。"},
		docs:   seoText{"APIドキュメント", "AI APIの接続、クライアントの設定、APIキーの管理方法をご案内します。"},
	},
	{
		code: "ko", tag: "ko", openGraph: "ko_KR",
		home:   seoText{"AI 모델 API 게이트웨이", "통합 API로 AI 모델에 연결하고 모델 요금과 연동 가이드를 확인하세요."},
		market: seoText{"모델 마켓", "AI 모델의 입력, 출력, 캐시 요금을 그룹별로 비교하세요."},
		docs:   seoText{"API 문서", "AI API 연동, 클라이언트 설정, API 키 관리 가이드를 확인하세요."},
	},
}
