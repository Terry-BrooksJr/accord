from datetime import UTC, datetime
from enum import Enum
from typing import Any
from uuid import UUID, uuid4

from sqlalchemy import Column
from sqlalchemy.types import JSON
from sqlmodel import Field, SQLModel, create_engine


class Provider(int, Enum):
    UNKNOWN = 0
    STRIPE = 1
    SHIPPO = 2
    ZENDESK = 3
    PLAID = 4
    OTHER = 5


class ReceiptStatus(str, Enum):
    ReceiptStatusAccepted = "ACCEPTED"
    ReceiptStatusDuplicate = "DUPLICATE"
    ReceiptStatusError = "ERROR"


class WebhookReceipt(SQLModel, table=True):
    __tablename__ = "webhook_receipts"

    id: UUID = Field(
        primary_key=True,
        default_factory=uuid4,
        nullable=False,
    )
    correlation_id: UUID = Field(index=True)
    provider: Provider
    provider_event_id: str | None = Field(default=None, index=True)
    event_type: str = Field(index=True, nullable=False)
    payload: dict[str, Any] = Field(nullable=False, sa_column=Column(JSON))
    payload_hash: str = Field(nullable=False, unique=True)
    received_at: datetime = Field(
        default_factory=lambda: datetime.now(UTC), nullable=False
    )
    status: ReceiptStatus = Field(nullable=False)
    duplicate: bool = Field(nullable=False, default=False)


db_url = "postgresql://python_agent:zhb*ycd0rvc5zmz9YUG@172.19.0.2:5432/accord"
ENGINE = create_engine(db_url, echo=True)


def create_db_and_tables():
    SQLModel.metadata.create_all(ENGINE)


if __name__ == "__main__":
    create_db_and_tables()
    print("Database and tables created (if they didn't exist)!")
