client<llm> OpenRouter {
  provider openai-generic
  options {
    base_url "https://openrouter.ai/api/v1"
    api_key env.OPENROUTER_API_KEY
    model "anthropic/claude-haiku-4.5"
  }
}

client<llm> OpenAI {
  provider openai
  retry_policy Exponential
  options {
    model "gpt-4o-mini"
    api_key env.OPENAI_API_KEY
  }
}

client<llm> Anthropic {
  provider anthropic
  retry_policy Exponential
  options {
    model "claude-haiku-4.5"
    api_key env.ANTHROPIC_API_KEY
  }
}

retry_policy Exponential {
  max_retries 2
  strategy {
    type exponential_backoff
    delay_ms 300
    multiplier 1.5
    max_delay_ms 10000
  }
}
