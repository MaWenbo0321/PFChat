package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DashScope API 配置
const (
	dashScopeDefaultAPIURL  = "https://llm-26cli7e69esmbtok.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	dashScopeGenerationPath = "/services/aigc/multimodal-generation/generation"
	defaultModel            = "qwen3.8-flash"
)

// DashScope 请求结构（原生 HTTP 调用格式）
type DashScopeRequest struct {
	Model      string              `json:"model"`
	Input      DashScopeInput      `json:"input"`
	Parameters DashScopeParameters `json:"parameters,omitempty"`
}

type DashScopeInput struct {
	Messages []DashScopeMessage `json:"messages"`
}

type DashScopeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type DashScopeContentPart struct {
	Text string `json:"text,omitempty"`
}

type DashScopeParameters struct {
	ResultFormat        string                   `json:"result_format,omitempty"`
	Temperature         float64                  `json:"temperature,omitempty"`
	MaxCompletionTokens int                      `json:"max_completion_tokens,omitempty"`
	ResponseFormat      *DashScopeResponseFormat `json:"response_format,omitempty"`
	EnableThinking      *bool                    `json:"enable_thinking,omitempty"`
	IncrementalOutput   *bool                    `json:"incremental_output,omitempty"`
	EnableSearch        *bool                    `json:"enable_search,omitempty"`
	SearchOptions       *DashScopeSearchOptions  `json:"search_options,omitempty"`
}

type DashScopeResponseFormat struct {
	Type string `json:"type"`
}

type DashScopeSearchOptions struct {
	ForcedSearch   bool   `json:"forced_search,omitempty"`
	EnableSource   bool   `json:"enable_source,omitempty"`
	EnableCitation bool   `json:"enable_citation,omitempty"`
	CitationFormat string `json:"citation_format,omitempty"`
	SearchStrategy string `json:"search_strategy,omitempty"`
}

type DashScopeStatusCode int

func (c *DashScopeStatusCode) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*c = 0
		return nil
	}
	var code int
	if err := json.Unmarshal(data, &code); err == nil {
		*c = DashScopeStatusCode(code)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		*c = 0
		return nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return err
	}
	*c = DashScopeStatusCode(parsed)
	return nil
}

// DashScope 响应结构（原生 HTTP 调用格式）
type DashScopeResponse struct {
	StatusCode DashScopeStatusCode `json:"status_code"`
	RequestID  string              `json:"request_id"`
	Code       string              `json:"code"`
	Message    string              `json:"message"`
	Output     struct {
		Text         string `json:"text,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
		SearchInfo   struct {
			SearchResults []DashScopeSearchResult `json:"search_results,omitempty"`
		} `json:"search_info,omitempty"`
		Choices []struct {
			FinishReason string                   `json:"finish_reason"`
			Message      DashScopeResponseMessage `json:"message"`
		} `json:"choices,omitempty"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type DashScopeSearchResult struct {
	Index    any    `json:"index,omitempty"`
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
	SiteName string `json:"site_name,omitempty"`
}

type DashScopeResponseMessage struct {
	Role             string          `json:"role"`
	Content          json.RawMessage `json:"content"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
}

// GrammarCheckResponse 语用失误检查响应结构 (LLM 返回的 JSON)
type GrammarCheckResponse struct {
	HasError            bool   `json:"has_error"`
	ConversationSummary string `json:"conversation_summary"`
	IntendedMeaning     string `json:"intended_meaning"`
	Suggestion          string `json:"suggestion"`
	Explanation         string `json:"explanation"`
	ErrorType           string `json:"error_type"`
	// 新增字段: LLM 返回的详细分析
	Impoliteness               bool   `json:"impoliteness"`
	LinguisticPragmaticFailure bool   `json:"linguistic_pragmatic_failure"`
	SocialPragmaticFailure     bool   `json:"social_pragmatic_failure"`
	OverallEvaluation          string `json:"overall_evaluation"` // "good" / "improvable" / "problematic"
}

// mapErrorType 根据 LLM 返回的详细分析结果映射到5种错误类型
func mapErrorType(result *GrammarCheckResponse) string {
	evaluation := strings.ToLower(strings.TrimSpace(result.OverallEvaluation))
	hasPragma := result.LinguisticPragmaticFailure
	hasSocio := result.SocialPragmaticFailure

	switch evaluation {
	case "problematic":
		// 严重情况
		if hasPragma && hasSocio {
			return ErrorTypeBothFailure // 语言语用失误和社会语用失误
		} else if hasSocio {
			return ErrorTypeSevereSociopragmatic // 严重社会语用失误
		} else if hasPragma {
			return ErrorTypeSeverePragmalinguistic // 严重语言语用失误
		}
	case "improvable":
		// 可改进情况
		if hasPragma && hasSocio {
			return ErrorTypeBothFailure
		} else if hasSocio {
			return ErrorTypeSociopragmatic // 社会语用失误
		} else if hasPragma {
			return ErrorTypePragmalinguistic // 语用语言失误
		}
	default:
		// 兜底: 根据失误类型判断
		if hasPragma && hasSocio {
			return ErrorTypeBothFailure
		} else if hasSocio {
			return ErrorTypeSociopragmatic
		} else if hasPragma {
			return ErrorTypePragmalinguistic
		}
	}
	return ""
}

// normalizeGrammarCheckResult reconciles occasionally contradictory LLM fields
// before the result is persisted or returned to the client.
func normalizeGrammarCheckResult(result *GrammarCheckResponse) {
	if result == nil {
		return
	}

	result.ConversationSummary = strings.TrimSpace(result.ConversationSummary)
	result.IntendedMeaning = sanitizeIntendedMeaning(result.IntendedMeaning)
	result.Suggestion = strings.TrimSpace(result.Suggestion)
	result.Explanation = strings.TrimSpace(result.Explanation)
	result.ErrorType = strings.TrimSpace(result.ErrorType)
	result.OverallEvaluation = strings.ToLower(strings.TrimSpace(result.OverallEvaluation))

	classifiedFailure := result.LinguisticPragmaticFailure || result.SocialPragmaticFailure
	evaluatedFailure := result.OverallEvaluation == "improvable" || result.OverallEvaluation == "problematic"
	legacyTypedFailure := IsValidErrorType(result.ErrorType) && result.ErrorType != ErrorTypeNoIssue
	result.HasError = result.HasError || classifiedFailure || evaluatedFailure || legacyTypedFailure

	// 兼容缺少布尔字段的旧响应，同时确保组合类型不会被降级成单一类型。
	if result.HasError && !classifiedFailure {
		switch result.ErrorType {
		case ErrorTypeSociopragmatic, ErrorTypeSevereSociopragmatic:
			result.SocialPragmaticFailure = true
		case ErrorTypeBothFailure:
			result.LinguisticPragmaticFailure = true
			result.SocialPragmaticFailure = true
		default:
			result.LinguisticPragmaticFailure = true
		}
	}

	if result.HasError {
		if result.OverallEvaluation != "improvable" && result.OverallEvaluation != "problematic" {
			switch result.ErrorType {
			case ErrorTypeSeverePragmalinguistic, ErrorTypeSevereSociopragmatic:
				result.OverallEvaluation = "problematic"
			default:
				result.OverallEvaluation = "improvable"
			}
		}
		result.ErrorType = mapErrorType(result)
		if result.ErrorType == "" {
			result.HasError = false
		}
	}

	if !result.HasError {
		result.ErrorType = ""
		result.ConversationSummary = ""
		result.IntendedMeaning = ""
		result.Suggestion = ""
		result.Explanation = ""
		result.OverallEvaluation = "good"
		result.Impoliteness = false
		result.LinguisticPragmaticFailure = false
		result.SocialPragmaticFailure = false
	}
}

// sanitizeIntendedMeaning keeps storage/UI labels out of the structured value.
// The prompt discourages labels, while this normalization is the final safeguard
// before per-turn or session-level feedback is persisted.
func sanitizeIntendedMeaning(value string) string {
	labels := []string{
		"说话者可能想表达",
		"说话者可能试图表达",
		"说话者似乎想表达",
		"说话者想表达",
		"说话者可能意图",
		"可能意图",
		"可能想表达",
		"Likely intended meaning",
		"Likely speaker intent",
		"The speaker may be trying to say",
		"The speaker may mean",
	}

	result := strings.TrimSpace(value)
	for {
		previous := result
		result = strings.TrimLeft(result, " \t\r\n>*_#-")
		for _, label := range labels {
			if len(result) >= len(label) && strings.EqualFold(result[:len(label)], label) {
				result = strings.TrimSpace(strings.TrimLeft(result[len(label):], " \t:：-—"))
				break
			}
		}
		if result == previous {
			return result
		}
	}
}

// appendEvaluationEvidenceGuard is shared by per-turn and whole-session
// evaluators so culture, attribution, and counterpart-blame rules cannot drift.
func appendEvaluationEvidenceGuard(sb *strings.Builder, isChinese bool) {
	if isChinese {
		sb.WriteString("硬性证据门槛：国家/地区、民族和母语只能用于核对身份与语言，不得作为礼貌、人格、动机、沟通距离或行为的解释证据。除非被分析原话本身含有群体概括，或对话明确陈述了某人的具体亲历/具体惯例，否则 summary、conversation_summary、explanation 和 suggestion 均不得提及国家文化，也不得写‘某国人/某国教师通常、往往、重视、倾向于……’。\n")
		sb.WriteString("归因门槛：只评价当前模式指定的学习者。对方的不耐烦、粗鲁或错误回复不能反向证明学习者有错；若学习者当前表达得体，不得因对方反应不佳而判错。\n")
		sb.WriteString("事实门槛：精确保留原文中的姓名、编号、日期和关键词；遇到多义词或非评估语言时优先原样引用，不得凭近形词或机器翻译替换。\n\n")
		return
	}
	sb.WriteString("Hard evidence gate: country/region, ethnicity, and native language may only verify identity and language. They are never evidence for politeness, personality, motives, social distance, or behavior. Unless the analyzed utterance itself makes a group claim or the transcript explicitly states a concrete personal experience/convention, summary, conversation_summary, explanation, and suggestion must not mention national culture or claim that people/teachers from a country usually, often, value, or tend to behave a certain way.\n")
	sb.WriteString("Attribution gate: evaluate only the learner designated by the session mode. A counterpart's impatient, rude, or mistaken response is not evidence that the learner erred; do not flag an appropriate learner utterance because the counterpart reacted poorly.\n")
	sb.WriteString("Fact gate: preserve names, identifiers, dates, and key source terms exactly. For ambiguous or foreign-language terms, quote the source form instead of replacing it with a look-alike or guessed translation.\n\n")
}

// buildCombinedPrompt 构建 PFChat 语用检测提示词。
func buildCombinedPrompt(history []Message, current Message, sender User, receiver User, session ConversationSession) string {
	var sb strings.Builder

	msgLang := detectMessageLanguage(current.Content)
	isChineseMsg := msgLang == "zh"
	targetLang := getLanguageFullName(session.TargetLanguage)
	suggestionLang := getSuggestionLanguageName(msgLang, session.TargetLanguage, isChineseMsg)
	analysisTarget := getAnalysisTargetName(current.Role, isChineseMsg)
	roleProfile := getSessionLLMRoleProfile(session)
	humanUser := sender
	if current.Role == ErrorSourceLLM {
		humanUser = receiver
	}
	llmCountry := getLLMPersonaNativeCountry(session, humanUser)
	llmNativeLang := getLLMPersonaNativeLanguage(session, humanUser)
	explanationCountry := sender.Country
	if current.Role == ErrorSourceLLM {
		explanationCountry = receiver.Country
	}
	explanationLang := getCountryLanguageName(explanationCountry, isChineseMsg)

	if isChineseMsg {
		sb.WriteString("你是 PFChat 项目中的跨文化语用学评估器。\n")
		sb.WriteString("PFChat 是一个语言学习聊天练习系统：用户与 LLM Bot 围绕指定关系和主题对话，系统只记录会影响交际效果的语用问题。\n")
		sb.WriteString(fmt.Sprintf("你的任务是判断“%s”在本次会话上下文中是否存在语用失误，并给出可直接展示在语用问题记录页的 JSON。\n\n", analysisTarget))

		sb.WriteString("会话上下文:\n")
		sb.WriteString(fmt.Sprintf("- 会话模式: %s\n", getSessionModePrompt(session.Mode, true)))
		sb.WriteString(fmt.Sprintf("- 目标/练习语言: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("- 对话关系: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("- 对话主题: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("- 当前分析对象: %s\n", analysisTarget))
		sb.WriteString(fmt.Sprintf("- LLM角色档案: %s，%d岁，%s；身份元数据（仅核对一致性）: 国家/地区 %s、母语/主要语言 %s；个体特点: %s；生活背景: %s\n\n",
			roleProfile.NameZH,
			roleProfile.Age,
			roleProfile.GenderZH,
			getCountryName(llmCountry),
			llmNativeLang,
			roleProfile.PersonalityZH,
			roleProfile.BackgroundZH,
		))

		sb.WriteString("判定标准:\n")
		sb.WriteString("- 只检测语用问题，不做普通语法、拼写、词汇或风格润色；只有这些问题改变礼貌、意图或关系处理时才标记。\n")
		sb.WriteString("- 根据对话关系判断得体性：陌生人、师生、同事/商务关系通常需要更高礼貌度；朋友、同学关系可更自然直接。\n")
		sb.WriteString("- 根据主题判断场景期待：学术、商务、旅行、文化交流和日常闲聊的表达规范不同。\n")
		sb.WriteString("- 分析 LLM Bot 时，优先从当前措辞、具体关系、角色的个人经历与个体弱点，以及可观察到的二语迁移解释问题；不得从国籍直接推断表达动机或性格。\n")
		sb.WriteString("- 国家/地区和母语只是背景信息，不是文化归因的充分证据。只有当前对话明确给出个人文化经历，或上下文提供了具体且可验证的文化惯例时，才可用‘可能与……有关’的有限表述；否则应在措辞、个人习惯、二语迁移和关系层面解释。\n")
		sb.WriteString("- 禁止使用‘某国文化就是……’‘某国人通常……’或‘这是某国文化中的习惯’等群体概括，不得根据国籍推断道德观、礼貌程度或行为动机。\n")
		sb.WriteString("- 如果当前消息本身在没有对话证据的情况下，用‘在某国这很礼貌’等说法概括整个国家/文化，应将其视为可能误导听者的社会语用问题，而不能因为角色来自该国家就放过。\n")
		sb.WriteString("- 只有角色配置的母语/主要语言与某种迁移分析相符，并且当前措辞提供了可观察的直接证据时，才能提出具体语言迁移；否则只能写‘可能的二语措辞/迁移’，不得使用‘中式英语’‘日式英语’等国别标签。\n")
		sb.WriteString("- 除非是在准确引用待分析原话，不得提及角色配置之外的其他国家、地区或母语。\n")
		sb.WriteString("- 修改无依据的群体概括时，suggestion 必须完全移除群体判断；把‘所有某国人’弱化成‘某国人通常/往往’仍不合格。\n")
		sb.WriteString("- 未知的动机、原因、责任、经历、期限和承诺也属于未知事实；不得在 intended_meaning、suggestion 或 explanation 中擅自补全。\n")
		sb.WriteString("- suggestion 必须保持原话的事实条件和立场，不能把一个立场或要求改写成推测原因。例如把‘不讨论’改成‘我没有时间讨论’会新增原因，禁止这样改写。\n")
		sb.WriteString("- 不要因为学习者表达不够地道就标记错误；只有可能造成冒犯、误解、请求/拒绝/感谢/道歉不当或关系失衡时才标记。\n")
		sb.WriteString("- 只评价当前分析对象，不把历史消息中的问题归因到当前消息。\n")
		sb.WriteString("- llm_l2 模式中，LLM Bot 会适当模拟二语学习者的语用问题；如果当前分析对象是 LLM Bot 回复，请识别其故意触发的语用失误。\n")
		sb.WriteString("- user_l2 模式中，重点评估用户用目标语言与母语者交流时的语用得体性。\n\n")
		sb.WriteString("分类一致性规则（强制）:\n")
		sb.WriteString("- 仅 linguistic_pragmatic_failure=true 时，error_type 只能是‘语用语言失误’或在 overall_evaluation=problematic 时使用‘严重语用语言失误’。\n")
		sb.WriteString("- 仅 social_pragmatic_failure=true 时，error_type 只能是‘社会语用失误’或在 overall_evaluation=problematic 时使用‘严重社会语用失误’。\n")
		sb.WriteString("- 两者均为 true 时，error_type 必须是‘语用语言失误和社会语用失误’；两者均为 false 时，has_error 必须为 false 且 error_type 留空。\n")
		sb.WriteString("- explanation 不得声称存在布尔字段未标记的错误类型；输出前必须检查布尔字段、error_type 和 explanation 完全一致。\n\n")
		if current.Role == ErrorSourceLLM {
			sb.WriteString("重要说明: 当前被分析对象是 LLM Bot 模拟的第二语言说话者。intended_meaning 必须单独写给 Human Listener，用中性、非断言语气说明说话者可能想完成的交际意图；该字段只能包含意图释义本身，不得带‘说话者可能想表达：’‘可能意图：’等字段名或前缀。LLM 作为学习者时不向 Human Listener 提供修改建议，因此 suggestion 必须为空字符串。explanation 再说明实际措辞可能让听者如何理解、为什么它是一个训练样例，以及听者可以观察到什么。不要用‘你应该……’对 LLM Bot 说教。如果 Human Listener 是当前语言的母语或熟练使用者，这项意图释义尤其重要；不得仅凭国家/地区推断其母语身份，无法确认时仍可提供释义，但应使用‘可能’‘看起来’等限定语。\n\n")
		} else {
			sb.WriteString("重要说明: 当前被分析对象是使用第二语言的人类用户。请提供可执行的 suggestion，但 intended_meaning 必须为空字符串；用户知道自己想表达什么，不需要系统替其猜测意图。\n\n")
		}

		sb.WriteString("字段含义:\n")
		sb.WriteString("- impoliteness: 当前消息是否明显不礼貌、冒犯、命令感过强或缺少必要缓和。\n")
		sb.WriteString("- linguistic_pragmatic_failure: 语言形式选择导致语用功能不当，例如请求、拒绝、道歉、感谢、称呼、缓和语或礼貌策略不合适。\n")
		sb.WriteString("- social_pragmatic_failure: 对社会关系、身份距离、权力差异、文化规范或场景期待判断不当。\n")
		sb.WriteString("- overall_evaluation: good 表示无明显问题；improvable 表示轻中度不合适但通常可修正；problematic 表示很可能冒犯或导致交际失败。\n\n")
	} else {
		sb.WriteString("You are the cross-cultural pragmatics evaluator inside PFChat.\n")
		sb.WriteString("PFChat is a language-learning chat practice system where a user and an LLM bot talk within a selected relationship and topic. The app only records pragmatic issues that affect communicative success.\n")
		sb.WriteString(fmt.Sprintf("Your task is to judge whether the %s has a pragmatic failure in this session context, then return JSON that can be shown directly in the pragmatic issue record page.\n\n", analysisTarget))

		sb.WriteString("Session context:\n")
		sb.WriteString(fmt.Sprintf("- Session mode: %s\n", getSessionModePrompt(session.Mode, false)))
		sb.WriteString(fmt.Sprintf("- Target/practice language: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("- Relationship: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("- Topic: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("- Current analysis target: %s\n", analysisTarget))
		sb.WriteString(fmt.Sprintf("- LLM role profile: %s, age %d, %s; identity metadata for consistency only: country/region %s and native/main language %s; individual traits: %s; life background: %s\n\n",
			roleProfile.NameEN,
			roleProfile.Age,
			roleProfile.GenderEN,
			getCountryName(llmCountry),
			llmNativeLang,
			roleProfile.PersonalityEN,
			roleProfile.BackgroundEN,
		))

		sb.WriteString("Evaluation rules:\n")
		sb.WriteString("- Detect pragmatic failures only, not ordinary grammar, spelling, vocabulary, or style issues unless they change politeness, intent, or relationship management.\n")
		sb.WriteString("- Judge appropriateness by relationship: strangers, teacher-student, colleague/business contexts usually require more politeness; friends and classmates can be more direct.\n")
		sb.WriteString("- Judge by topic: academic, business, travel, cultural exchange, and daily chat contexts have different expectations.\n")
		sb.WriteString("- When analyzing the LLM Bot, explain problems first through the current wording, relationship, the role's specific personal experience and individual limitations, and observable L2 transfer. Never infer motive or personality directly from nationality.\n")
		sb.WriteString("- Country/region and native language are context, not sufficient evidence for cultural attribution. Mention culture only when the conversation states a personal cultural experience or the context supplies a specific, verifiable convention; use bounded wording such as ‘may be related to’. Otherwise explain the issue at the wording, individual-habit, L2-transfer, and relationship levels.\n")
		sb.WriteString("- Do not make group claims such as ‘people from X usually...’ or ‘this is an X-cultural habit’, and never infer morality, politeness, or behavioral motives from nationality.\n")
		sb.WriteString("- If the current message itself makes an unsupported national or cultural generalization such as ‘In country X, this is polite’, treat it as a potential sociopragmatic problem that can mislead the listener; do not excuse it because the role comes from that country.\n")
		sb.WriteString("- Name a specific language transfer only when it matches the role's configured native/main language and the current wording supplies direct observable evidence. Otherwise use a neutral phrase such as ‘possible L2 wording/transfer’; never label it ‘Chinese English’, ‘Japanese English’, or another national variety.\n")
		sb.WriteString("- Do not mention a country, region, or native language different from the configured role unless accurately quoting the source message.\n")
		sb.WriteString("- A suggestion that repairs an unsupported group claim must remove the group judgment entirely. Changing ‘all people from X’ to ‘people from X often/usually’ is still unacceptable.\n")
		sb.WriteString("- Unknown motives, causes, responsibility, experiences, deadlines, and commitments are also unknown facts. Do not fill them in within intended_meaning, suggestion, or explanation.\n")
		sb.WriteString("- The suggestion must preserve the source message's truth conditions and stance; never replace a stance or request with an inferred reason. For example, changing ‘No discussion’ to ‘I do not have time to discuss’ invents a cause and is forbidden.\n")
		sb.WriteString("- Do not flag a message merely because it is non-native or not idiomatic; flag only likely offense, misunderstanding, inappropriate requests/refusals/thanks/apologies, or relationship mismatch.\n")
		sb.WriteString("- Evaluate only the current analysis target; do not attribute issues in previous messages to the current message.\n")
		sb.WriteString("- In llm_l2 mode, the LLM bot may appropriately simulate L2 pragmatic problems; if the current analysis target is an LLM Bot reply, identify the intentionally triggered pragmatic failure.\n")
		sb.WriteString("- In user_l2 mode, focus on whether the user's target-language message is pragmatically appropriate for a native-speaker interlocutor.\n\n")
		sb.WriteString("Mandatory classification consistency rules:\n")
		sb.WriteString("- If only linguistic_pragmatic_failure=true, error_type must be ‘语用语言失误’, or ‘严重语用语言失误’ only when overall_evaluation=problematic.\n")
		sb.WriteString("- If only social_pragmatic_failure=true, error_type must be ‘社会语用失误’, or ‘严重社会语用失误’ only when overall_evaluation=problematic.\n")
		sb.WriteString("- If both are true, error_type must be ‘语用语言失误和社会语用失误’. If both are false, has_error must be false and error_type must be empty.\n")
		sb.WriteString("- The explanation must not claim an error category whose boolean is false. Before output, verify that the booleans, error_type, and explanation agree exactly.\n\n")
		if current.Role == ErrorSourceLLM {
			sb.WriteString("Important: the current target is an LLM Bot simulating an L2 speaker. Write intended_meaning as a separate field for the Human Listener, neutrally and tentatively paraphrasing the communicative intention the speaker may have meant. This field must contain only the paraphrase itself, never a label or prefix such as ‘Likely intended meaning:’. When the LLM is the learner, do not give the Human Listener a revision, so suggestion must be an empty string. Use explanation only for how the actual wording may be interpreted, why it works as a training sample, and what the listener can observe. Do not lecture the LLM Bot with ‘you should...’. This intended-meaning paraphrase is especially important when the Human Listener is a native or proficient speaker of the current language. Never infer native-speaker status from country/region alone; if proficiency is uncertain, still offer the paraphrase using qualifiers such as ‘may mean’ or ‘appears to be trying to’.\n\n")
		} else {
			sb.WriteString("Important: the current target is a human user communicating in an L2. Provide an actionable suggestion, but intended_meaning must be an empty string; the user already knows their own intent and the system must not guess it for them.\n\n")
		}

		sb.WriteString("Field meanings:\n")
		sb.WriteString("- impoliteness: whether the current message is clearly rude, offensive, too commanding, or lacks necessary mitigation.\n")
		sb.WriteString("- linguistic_pragmatic_failure: whether the wording/form choice makes the speech act pragmatically inappropriate, such as requests, refusals, apologies, thanks, address terms, mitigation, or politeness strategies.\n")
		sb.WriteString("- social_pragmatic_failure: whether the message misjudges social relationship, distance, power, cultural norms, or situational expectations.\n")
		sb.WriteString("- overall_evaluation: good means no clear issue; improvable means mild/moderate inappropriateness; problematic means likely offense or communication breakdown.\n\n")
	}
	appendEvaluationEvidenceGuard(&sb, isChineseMsg)

	if len(history) > 0 {
		if isChineseMsg {
			sb.WriteString("最近聊天记录（从旧到新）:\n")
		} else {
			sb.WriteString("Recent chat history (oldest first):\n")
		}
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			var senderName string
			if msg.Role == "user" || (msg.Role == "" && msg.Sender.Role != RoleBot) {
				if isChineseMsg {
					senderName = "用户"
				} else {
					senderName = "User"
				}
			} else {
				senderName = "LLM Bot"
			}
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", senderName, msg.Content))
		}
		sb.WriteString("\n")
	}

	if isChineseMsg {
		sb.WriteString(fmt.Sprintf("当前待分析的%s: %q\n\n", analysisTarget, current.Content))
	} else {
		sb.WriteString(fmt.Sprintf("Current %s to analyze: %q\n\n", analysisTarget, current.Content))
	}

	if isChineseMsg {
		sb.WriteString("请以以下 JSON 格式分析（不要包含 markdown 代码块，只输出纯 JSON）:\n")
	} else {
		sb.WriteString("Analyze and respond in the following JSON format (no markdown code blocks, pure JSON only):\n")
	}

	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"impoliteness\": true/false,\n")
	sb.WriteString("  \"linguistic_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"social_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"overall_evaluation\": \"good\"/\"improvable\"/\"problematic\",\n")
	sb.WriteString("  \"error_type\": \"语用语言失误/社会语用失误/严重语用语言失误/严重社会语用失误/语用语言失误和社会语用失误\",\n")
	sb.WriteString("  \"conversation_summary\": \"about five concise sentences describing the relevant conversation context\",\n")
	if current.Role == ErrorSourceLLM {
		sb.WriteString("  \"intended_meaning\": \"tentative paraphrase of what the LLM speaker likely meant, in the human listener's language\",\n")
		sb.WriteString("  \"suggestion\": \"\",\n")
	} else {
		sb.WriteString("  \"intended_meaning\": \"\",\n")
		sb.WriteString("  \"suggestion\": \"single revised sentence in the SAME language as the original message\",\n")
	}
	sb.WriteString("  \"explanation\": \"brief reason and improvement advice\"\n")
	sb.WriteString("}\n\n")

	if isChineseMsg {
		sb.WriteString("如果没有语用失误，返回:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"error_type\": \"\", \"conversation_summary\": \"\", \"intended_meaning\": \"\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		sb.WriteString(fmt.Sprintf("conversation_summary 必须使用%s，用大约5个简洁句子概括与本次错误直接相关的对话：说明主题、双方关系、双方大致或准确说了什么，以及当前问题出现前后的语境。只能使用聊天记录和会话字段中的事实，不得补写事件、原因或动机。\n", explanationLang))
		if current.Role == ErrorSourceLLM {
			sb.WriteString(fmt.Sprintf("输出语言要求: intended_meaning 和 explanation 必须使用%s；suggestion 必须为空字符串。\n", explanationLang))
		} else {
			sb.WriteString(fmt.Sprintf("输出语言要求: explanation 必须使用%s；intended_meaning 必须为空字符串；suggestion 必须使用%s，并且只包含一个最佳修改句。suggestion 只能调整措辞，不得新增或改变原消息中的原因、责任、时间、承诺或其他事实。\n", explanationLang, suggestionLang))
		}
		sb.WriteString("如果 has_error 为 false，conversation_summary、intended_meaning、suggestion 和 explanation 必须为空字符串。输出前核对人物国家/地区、母语、关系和已知事实，并确保布尔分类、error_type、suggestion 与 explanation 相互一致。\n")
	} else {
		sb.WriteString("If no pragmatic failure is detected, return:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"error_type\": \"\", \"conversation_summary\": \"\", \"intended_meaning\": \"\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		sb.WriteString(fmt.Sprintf("conversation_summary must be in %s and use about five concise sentences to summarize only the conversation relevant to this issue: include the topic, relationship, what each person said approximately or exactly, and the context immediately around the problem. Use only facts present in the chat and session fields; never invent events, causes, or motives.\n", explanationLang))
		if current.Role == ErrorSourceLLM {
			sb.WriteString(fmt.Sprintf("Output language requirements: intended_meaning and explanation must be in %s; suggestion must be an empty string.\n", explanationLang))
		} else {
			sb.WriteString(fmt.Sprintf("Output language requirements: explanation must be in %s; intended_meaning must be an empty string; suggestion must be in %s and contain only one best revised sentence. The suggestion may improve wording only and must not add or change causes, responsibility, timing, commitments, or any other fact from the original message.\n", explanationLang, suggestionLang))
		}
		sb.WriteString("If has_error is false, conversation_summary, intended_meaning, suggestion, and explanation must be empty strings. Before output, verify the role's country/region, native language, relationship, and known facts, then ensure the classification booleans, error_type, suggestion, and explanation are mutually consistent.\n")
	}

	return sb.String()
}

func getAnalysisTargetName(role string, isChinese bool) string {
	if role == ErrorSourceLLM {
		if isChinese {
			return "LLM Bot 当前回复"
		}
		return "LLM Bot reply"
	}
	if isChinese {
		return "用户当前消息"
	}
	return "user message"
}
func getSessionModePrompt(mode string, isChinese bool) string {
	switch mode {
	case ModeUserL2:
		if isChinese {
			return "user_l2，用户使用目标语言练习，LLM Bot 作为目标语言母语者自然回应"
		}
		return "user_l2, the user practices in the target language and the LLM bot replies as a native speaker"
	case ModeLLML2:
		if isChinese {
			return "llm_l2，LLM Bot 使用目标语言模拟二语学习者，用户通常作为母语者参与对话"
		}
		return "llm_l2, the LLM bot simulates an L2 learner in the target language and the user usually participates as a native speaker"
	default:
		if isChinese {
			return "未知模式，按普通跨文化聊天练习处理"
		}
		return "unknown mode, treat as a regular cross-cultural chat practice session"
	}
}

func getSuggestionLanguageName(msgLang string, targetLang string, isChinese bool) string {
	switch msgLang {
	case "zh":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "ja":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "ko":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	default:
		return getTargetLanguagePromptName(targetLang, isChinese)
	}
}

func getTargetLanguagePromptName(lang string, isChinese bool) string {
	switch strings.ToUpper(lang) {
	case "ZH":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "JP":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "KR":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	case "FR":
		if isChinese {
			return "法语"
		}
		return "French"
	case "DE":
		if isChinese {
			return "德语"
		}
		return "German"
	default:
		if isChinese {
			return "英语"
		}
		return "English"
	}
}

func getCountryLanguageName(country string, isChinese bool) string {
	switch country {
	case "CN", "TW", "HK", "SG":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "JP":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "KR":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	case "FR":
		if isChinese {
			return "法语"
		}
		return "French"
	case "DE":
		if isChinese {
			return "德语"
		}
		return "German"
	default:
		if isChinese {
			return "英语"
		}
		return "English"
	}
}

func getDashScopeAPIKey() string {
	if apiKey := strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY")); apiKey != "" {
		return apiKey
	}
	return ""
}

func getDashScopeModel() string {
	if model := strings.TrimSpace(os.Getenv("DASHSCOPE_MODEL")); model != "" {
		return model
	}
	return defaultModel
}

func getDashScopeGenerationURL() string {
	if apiURL := strings.TrimSpace(os.Getenv("DASHSCOPE_API_URL")); apiURL != "" {
		return strings.TrimRight(apiURL, "/")
	}

	baseURL := strings.TrimSpace(os.Getenv("DASHSCOPE_BASE_URL"))
	if baseURL == "" {
		return dashScopeDefaultAPIURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/generation") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/api/v1") {
		return baseURL + dashScopeGenerationPath
	}
	return baseURL + "/api/v1" + dashScopeGenerationPath
}

func boolPtr(v bool) *bool {
	return &v
}

func newDashScopeTextMessage(role string, content string) DashScopeMessage {
	return DashScopeMessage{
		Role: role,
		Content: []DashScopeContentPart{
			{Text: content},
		},
	}
}

// makeDashScopeRequest 发起非流式 DashScope 原生 HTTP 请求（可复用）。
func makeDashScopeRequest(apiKey string, reqBody DashScopeRequest) (*DashScopeResponse, error) {
	return makeDashScopeHTTPRequest(apiKey, reqBody, false, 30*time.Second)
}

// makeDashScopeStreamingRequest 发起 SSE 流式请求。多模态模型的联网搜索必须使用此模式。
func makeDashScopeStreamingRequest(apiKey string, reqBody DashScopeRequest) (*DashScopeResponse, error) {
	return makeDashScopeHTTPRequest(apiKey, reqBody, true, 45*time.Second)
}

func makeDashScopeHTTPRequest(apiKey string, reqBody DashScopeRequest, streaming bool, timeout time.Duration) (*DashScopeResponse, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", getDashScopeGenerationURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if streaming {
		req.Header.Set("X-DashScope-SSE", "enable")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil {
			return nil, fmt.Errorf("API 返回错误 %d，且读取错误内容失败: %v", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}
	if streaming {
		return parseDashScopeSSE(resp.Body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var dashResp DashScopeResponse
	if err := json.Unmarshal(body, &dashResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if dashResp.StatusCode != 0 && dashResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误 %d: %s %s", dashResp.StatusCode, dashResp.Code, dashResp.Message)
	}

	return &dashResp, nil
}

// parseDashScopeSSE 将 incremental_output 模式下的增量内容合并为普通响应，
// 从而让上层生成逻辑无需同时维护两套响应处理代码。
func parseDashScopeSSE(reader io.Reader) (*DashScopeResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)

	var merged DashScopeResponse
	var textBuilder strings.Builder
	seenEvent := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var event DashScopeResponse
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil, fmt.Errorf("解析 SSE 响应失败: %w", err)
		}
		seenEvent = true
		if event.StatusCode != 0 && event.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API 返回错误 %d: %s %s", event.StatusCode, event.Code, event.Message)
		}
		if event.Code != "" {
			return nil, fmt.Errorf("API 返回错误: %s %s", event.Code, event.Message)
		}

		textBuilder.WriteString(getDashScopeResponseText(&event))
		if event.RequestID != "" {
			merged.RequestID = event.RequestID
		}
		if len(event.Output.SearchInfo.SearchResults) > 0 {
			merged.Output.SearchInfo.SearchResults = event.Output.SearchInfo.SearchResults
		}
		if event.Output.FinishReason != "" {
			merged.Output.FinishReason = event.Output.FinishReason
		}
		if len(event.Output.Choices) > 0 && event.Output.Choices[0].FinishReason != "" {
			merged.Output.FinishReason = event.Output.Choices[0].FinishReason
		}
		if event.Usage.TotalTokens > 0 {
			merged.Usage = event.Usage
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 SSE 响应失败: %w", err)
	}
	if !seenEvent {
		return nil, fmt.Errorf("SSE 响应中没有有效事件")
	}

	merged.Output.Text = textBuilder.String()
	return &merged, nil
}

func getDashScopeResponseText(resp *DashScopeResponse) string {
	if resp == nil {
		return ""
	}
	if len(resp.Output.Choices) > 0 {
		text := extractDashScopeContentText(resp.Output.Choices[0].Message.Content)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return resp.Output.Text
}

func extractDashScopeContentText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err == nil {
		var sb strings.Builder
		for _, part := range parts {
			if value, ok := part["text"].(string); ok {
				sb.WriteString(value)
			}
		}
		return sb.String()
	}

	var part map[string]any
	if err := json.Unmarshal(raw, &part); err == nil {
		if value, ok := part["text"].(string); ok {
			return value
		}
	}

	return string(raw)
}

// callDashScopeAPI 调用 DashScope API 进行语用检查
func callDashScopeAPI(prompt string) (*GrammarCheckResponse, error) {
	reqBody := DashScopeRequest{
		Model: getDashScopeModel(),
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				newDashScopeTextMessage("user", prompt),
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat:        "message",
			Temperature:         0.3,
			MaxCompletionTokens: 640,
			ResponseFormat:      &DashScopeResponseFormat{Type: "json_object"},
			EnableThinking:      boolPtr(false),
		},
	}

	dashResp, err := makeDashScopeRequest(getDashScopeAPIKey(), reqBody)
	if err != nil {
		return nil, err
	}

	// 提取响应文本
	responseText := getDashScopeResponseText(dashResp)
	if strings.TrimSpace(responseText) == "" {
		return nil, fmt.Errorf("empty pragmatic check response")
	}

	// 提取 JSON
	jsonStr := extractJSON(responseText)
	log.Printf("LLM Response JSON: %s", jsonStr)

	var result GrammarCheckResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Printf("JSON parse error: %v, raw: %s", err, jsonStr)
		return nil, fmt.Errorf("parse pragmatic check response: %w", err)
	}

	normalizeGrammarCheckResult(&result)

	return &result, nil
}

func callDashScopeJSON(prompt string, target any, maxCompletionTokens int) error {
	return callDashScopeJSONWithTemperature(prompt, target, maxCompletionTokens, 0.2)
}

func callDashScopeJSONWithTemperature(prompt string, target any, maxCompletionTokens int, temperature float64) error {
	reqBody := DashScopeRequest{
		Model: getDashScopeModel(),
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				newDashScopeTextMessage("user", prompt),
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat:        "message",
			Temperature:         temperature,
			MaxCompletionTokens: maxCompletionTokens,
			ResponseFormat:      &DashScopeResponseFormat{Type: "json_object"},
			EnableThinking:      boolPtr(false),
		},
	}

	dashResp, err := makeDashScopeRequest(getDashScopeAPIKey(), reqBody)
	if err != nil {
		return err
	}

	responseText := getDashScopeResponseText(dashResp)
	if strings.TrimSpace(responseText) == "" {
		return fmt.Errorf("empty JSON response from LLM")
	}

	jsonStr := extractJSON(responseText)
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("解析 JSON 响应失败: %v, raw: %s", err, jsonStr)
	}
	return nil
}
func extractJSON(text string) string {
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return text
	}

	return text[start : end+1]
}

func getCountryName(countryCode string) string {
	countryMap := map[string]string{
		"CN":    "中国(China)",
		"US":    "美国(USA)",
		"GB":    "英国(UK)",
		"JP":    "日本(Japan)",
		"KR":    "韩国(Korea)",
		"FR":    "法国(France)",
		"DE":    "德国(Germany)",
		"CA":    "加拿大(Canada)",
		"AU":    "澳大利亚(Australia)",
		"NG":    "尼日利亚(Nigeria)",
		"BR":    "巴西(Brazil)",
		"ZA":    "南非(South Africa)",
		"IN":    "印度(India)",
		"MX":    "墨西哥(Mexico)",
		"MN":    "蒙古(Mongolia)",
		"MY":    "马来西亚(Malaysia)",
		"SG":    "新加坡(Singapore)",
		"OTHER": "其他(Other)",
	}
	if name, ok := countryMap[countryCode]; ok {
		return name
	}
	return countryCode
}

// ============================================================================
// 【说明】buildCombinedPrompt 基于消息实际语言和会话上下文构造提示词。
//   - suggestion 字段：与消息内容同语言。
//   - explanation 字段：使用发送者的主要语言，方便学习者理解。
//   - 判定标准：贴合 PFChat 的会话模式、目标语言、关系、主题和五类语用失误。
// ============================================================================

// detectMessageLanguage 检测消息的主要语言。
// 返回 "zh"（中文）、"ja"（日文）、"ko"（韩文）或 "en"（英文及其他）。
func detectMessageLanguage(content string) string {
	chineseCount := 0
	japaneseCount := 0
	koreanCount := 0
	totalLetters := 0

	for _, r := range content {
		if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) {
			// CJK 汉字（中文为主，日文汉字也在此范围，但平假名/片假名更有区分力）
			chineseCount++
			totalLetters++
		} else if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			// 平假名或片假名 → 日文特征
			japaneseCount++
			totalLetters++
		} else if r >= 0xAC00 && r <= 0xD7AF {
			// 韩文音节
			koreanCount++
			totalLetters++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			totalLetters++
		}
	}

	if totalLetters == 0 {
		return "en"
	}

	// 日文优先判断（因为日文也含汉字，但有假名）
	if japaneseCount > 0 {
		return "ja"
	}
	if koreanCount > 0 {
		return "ko"
	}
	threshold := totalLetters / 4 // 超过1/4汉字视为中文
	if chineseCount > threshold {
		return "zh"
	}
	return "en"
}
