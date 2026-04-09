from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    anthropic_api_key: str
    bridge_url: str = "http://localhost:8080"

    model_config = {"env_file": ".env"}


settings = Settings()
