package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

type BotCategory = fastly.BotCategory

const (
	// BotCategoryNone indicates bot detection was not executed, or no bot was detected.
	BotCategoryNone = fastly.BotCategoryNone

	// BotCategorySuspected is for a suspected bot.
	BotCategorySuspected = fastly.BotCategorySuspected

	// BotCategoryAccessibility is for tools that make content accessible (e.g., screen readers).
	BotCategoryAccessibility = fastly.BotCategoryAccessibility

	// BotCategoryAICrawler is for crawlers used for training AIs and LLMs, generally used for building AI models or indexes.
	BotCategoryAICrawler = fastly.BotCategoryAICrawler

	// BotCategoryAIFetcher is for fetchers used by AIs and LLMs for enriching results in response to a user query.
	BotCategoryAIFetcher = fastly.BotCategoryAIFetcher

	// BotCategoryContentFetcher is for tools that extract content from websites to be used elsewhere.
	BotCategoryContentFetcher = fastly.BotCategoryContentFetcher

	// BotCategoryMonitoringSiteTools is for tools that access your website to monitor things like performance, uptime, and proving domain control.
	BotCategoryMonitoringSiteTools = fastly.BotCategoryMonitoringSiteTools

	// BotCategoryOnlineMarketing is for crawlers from online marketing platforms (e.g., Facebook, Pinterest).
	BotCategoryOnlineMarketing = fastly.BotCategoryOnlineMarketing

	// BotCategoryPagePreview is for tools that access your website to show a preview of the page in other online services and social media platforms.
	BotCategoryPagePreview = fastly.BotCategoryPagePreview

	// BotCategoryPlatformIntegrations is for integration with other platforms by accessing the website's API, notably Webhooks.
	BotCategoryPlatformIntegrations = fastly.BotCategoryPlatformIntegrations

	// BotCategoryResearch is for commercial and academic tools that collect and analyze data for research purposes.
	BotCategoryResearch = fastly.BotCategoryResearch

	// BotCategorySearchEngineCrawler is for crawlers that index your website for search engines.
	BotCategorySearchEngineCrawler = fastly.BotCategorySearchEngineCrawler

	// BotCategorySearchEngineSpecialization is for tools that support search engine optimization tasks (e.g., link analysis, ranking).
	BotCategorySearchEngineSpecialization = fastly.BotCategorySearchEngineSpecialization

	// BotCategorySecurityTools is for security analysis tools that inspect your website for vulnerabilities, misconfigurations and other security features.
	BotCategorySecurityTools = fastly.BotCategorySecurityTools

	// BotCategoryUnknown indicates the detected bot belongs to a category not recognized by this SDK version.
	BotCategoryUnknown = fastly.BotCategoryUnknown
)

type BotDetectionResult struct {
	// Analyzed indicates if the request was analyzed by the bot detection framework.
	Analyzed bool

	// Detected indicates if a bot was detected.
	Detected bool

	// Name is string identifying the specific bot detected (e.g., `GoogleBot`, `GPTBot`, `Bingbot`).
	// Returns the empty string if bot detection was not executed or no bot was detected.
	//
	// Note: String values may change over time. Use this for logging or informational purposes.
	// For conditional logic, use CategoryKind.
	Name string

	// Category is a string indicating the type of bot detected (e.g., `SEARCH-ENGINE-CRAWLER`, `AI-CRAWLER`,
	// `SUSPECTED-BOT`).
	//
	// Note: String values may change over time. Use this for logging or informational purposes.
	// For conditional logic, use [`get_bot_category_kind()`][Self::get_bot_category_kind].
	Category string

	// An enum uniquely identifying the type of bot detected.
	CategoryKind BotCategory

	// Verified is whether the detected bot is a verified bot.
	Verfied bool
}

func (r *Request) BotDetection() (*BotDetectionResult, error) {
	var result BotDetectionResult

	var err error
	if result.Analyzed, err = r.downstream.req.DownstreamBotAnalyzed(); err != nil {
		return nil, err
	}

	// Didn't analyze the request?  Nothing else to do.
	if !result.Analyzed {
		return &result, nil
	}

	if result.Detected, err = r.downstream.req.DownstreamBotDetected(); err != nil {
		return nil, err
	}

	// Request wasn't detected as a bot?  Nothing to fill in.
	if !result.Detected {
		return &result, nil
	}

	if result.Name, err = r.downstream.req.DownstreamBotName(); err != nil {
		return nil, err
	}

	if result.Category, err = r.downstream.req.DownstreamBotCategory(); err != nil {
		return nil, err
	}

	if result.Verfied, err = r.downstream.req.DownstreamBotVerified(); err != nil {
		return nil, err
	}

	return &result, nil
}
