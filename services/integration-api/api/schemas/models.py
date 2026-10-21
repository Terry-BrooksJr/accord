from datetime import datetime
from enum import Enum
from typing import Any
from uuid import UUID

from sqlalchemy import Column, Text
from sqlalchemy.dialects.postgresql import INET, TEXT
from sqlalchemy.types import JSONB
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


class RejectionReason(str, Enum):
    MissingSignature = "NO_SIGNATURE"
    InvaildSignature = "BAD_SIGNATURE"
    ExpiredSignature = "OLD_SIGNATURE"
    MalformedSignature = "MISHAPEN_SIGNATURE"
    InvalidPayload = "BAD_PAYLOAD"
    PayloadTooLarge = "OVERSIZED_PAYLOAD"
    UnsupportedContentType = "BAD_CONTENT_TYPE"
    BodyReadFailure = "UNABLE_TO_PARSE"


class WebhookReceipt(SQLModel, table=True):
    __tablename__ = "webhook_receipts"

    id: UUID = Field(
        primary_key=True,
    )
    correlation_id: UUID
    provider: Provider = Field(nullable=False, index=True)
    provider_event_id: str | None = Field(default=None, index=True)
    event_type: str = Field(index=True, nullable=False)
    rawpayload: dict[str, Any] = Field(sa_column=Column(JSONB))
    payload_hash: str = Field(nullable=False, unique=True)
    received_at: datetime
    provider_created_at: datetime
    status: ReceiptStatus = Field(nullable=False)
    is_duplicate: bool = Field(nullable=False, default=False)


class WebhookRejection(SQLModel, table=True):
    __tablename__ = "webhook_rejections"
    id: UUID = Field(
        primary_key=True,
    )
    correlation_id: UUID
    provider: Provider = Field(nullable=False, index=True)
    received_at: datetime
    rejection_reason: RejectionReason
    request_path: str
    request_headers: dict[str, Any] = Field(sa_column=Column(JSONB))
    is_payload_truncated: bool
    sanitized_payload: str = Field(sa_type=Text, default=None)
    payload_sha256: str
    payload_size_bytes: int
    payload_format: str
    sanitizer_version: float
    source_ip: str = Field(sa_type=INET)
    rejection_details: str = Field(sa_type=TEXT, default=None)


db_url = "postgresql://python_agent:zhb*ycd0rvc5zmz9YUG@172.19.0.2:5432/accord"
ENGINE = create_engine(db_url, echo=True)


def create_db_and_tables():
    SQLModel.metadata.create_all(ENGINE)


if __name__ == "__main__":
    create_db_and_tables()
    print("Database and tables created (if they didn't exist)!")
