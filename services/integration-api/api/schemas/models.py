import os
from datetime import datetime
from enum import Enum, StrEnum
from typing import Any
from uuid import UUID

import dotenv
from sqlalchemy import Column, DateTime, Integer, LargeBinary, String
from sqlalchemy.dialects.postgresql import INET, TEXT
from sqlalchemy.types import JSON
from sqlmodel import Field, SQLModel, create_engine

dotenv.load_dotenv("/workspaces/accord/.env")


class Provider(int, Enum):
    """
    Identifies the service provider associated with a webhook.
    Each value represents a supported provider or an unknown or other provider category.
    """

    UNKNOWN = 0
    STRIPE = 1
    SHIPPO = 2
    ZENDESK = 3
    PLAID = 4
    OTHER = 5


class ReceiptStatus(StrEnum):
    """
    Represents the processing status of a webhook receipt.
    Each value identifies whether the receipt was accepted, duplicated, or encountered an error.
    """

    ReceiptStatusAccepted = "ACCEPTED"
    ReceiptStatusDuplicate = "DUPLICATE"
    ReceiptStatusError = "ERROR"


class RejectionReason(StrEnum):
    """
    Represents the reason a webhook request was rejected. Each value identifies the specific validation or
    processing issue that caused the rejection.
    """

    MissingSignature = "NO_SIGNATURE"
    InvaildSignature = "BAD_SIGNATURE"
    ExpiredSignature = "OLD_SIGNATURE"
    MalformedSignature = "MISHAPEN_SIGNATURE"
    InvalidPayload = "BAD_PAYLOAD"
    PayloadTooLarge = "OVERSIZED_PAYLOAD"
    UnsupportedContentType = "BAD_CONTENT_TYPE"
    BodyReadFailure = "UNABLE_TO_PARSE"


class WebhookReceipt(SQLModel, table=True):
    """
    Represents a webhook event received from an external service.
    Each record stores the event's provider, payload, timing information, and processing status.
    """

    __tablename__ = "webhook_receipts"

    id: UUID = Field(
        primary_key=True,
    )
    correlation_id: UUID
    provider: int = Field(sa_column=Column(Integer, nullable=False, index=True))
    provider_event_id: str | None = Field(default=None, index=True)
    event_type: str = Field(
        sa_column=Column(
            nullable=False,
            index=True,
        )
    )
    payload: dict[str, Any] = Field(sa_column=Column(JSON))
    payload_hash: str = Field(sa_column=Column(nullable=False, unique=True))
    received_at: datetime = Field(
        sa_column=Column(DateTime(timezone=True), nullable=False)
    )
    provider_created_at: datetime = Field(
        sa_column=Column(DateTime(timezone=True), nullable=False)
    )
    status: ReceiptStatus = Field(sa_column=Column(String, nullable=False))
    is_duplicate: bool = Field(nullable=False, default=False)


class WebhookRejection(SQLModel, table=True):
    __tablename__ = "webhook_rejections"
    id: UUID = Field(
        primary_key=True,
    )
    correlation_id: UUID = Field(index=True)
    provider: int = Field(sa_column=Column(Integer, nullable=False, index=True))
    received_at: datetime = Field(index=True)
    rejection_reason: str = Field(sa_column=Column(String, nullable=False))
    request_path: str
    request_headers: dict[str, list[str]] = Field(
        sa_column=Column(JSON, nullable=False)
    )
    is_payload_truncated: bool
    captured_payload: bytes | None = Field(sa_column=Column(LargeBinary, default=None))

    payload_sha256: str | None
    source_ip: str = Field(sa_column=Column(INET, nullable=False))
    rejection_details: str | None = Field(sa_column=Column(TEXT, default=None))


db_url = os.environ["PYTHON_POSTGRES_DB_CONNECTION_STRING"]
ENGINE = create_engine(db_url, echo=True)


def create_db_and_tables():
    SQLModel.metadata.create_all(ENGINE)


if __name__ == "__main__":
    create_db_and_tables()
    print("Database and tables created (if they didn't exist)!")
