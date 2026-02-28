"""Shared fixtures for ml-worker tests."""

from __future__ import annotations

from unittest.mock import MagicMock

import numpy as np
import pytest

from ml_worker.proto import chat_pb2, embed_pb2


# ---------------------------------------------------------------------------
# gRPC context mock
# ---------------------------------------------------------------------------

@pytest.fixture()
def grpc_context():
    """Return a mock gRPC ServicerContext."""
    ctx = MagicMock()
    ctx.abort = MagicMock(side_effect=Exception("grpc abort"))
    return ctx


# ---------------------------------------------------------------------------
# Chat fixtures
# ---------------------------------------------------------------------------

@pytest.fixture()
def chat_request():
    """Minimal ChatRequest."""
    return chat_pb2.ChatRequest(message="Hello")


@pytest.fixture()
def chat_request_full():
    """ChatRequest with all fields populated."""
    return chat_pb2.ChatRequest(
        message="Tell me a joke",
        context=["previous context"],
        system_prompt="You are helpful.",
        temperature=0.7,
        model="llama3",
        max_tokens=100,
    )


# ---------------------------------------------------------------------------
# Embed fixtures
# ---------------------------------------------------------------------------

@pytest.fixture()
def embed_request():
    return embed_pb2.EmbedRequest(text="hello world")


@pytest.fixture()
def embed_batch_request():
    return embed_pb2.EmbedBatchRequest(texts=["hello", "world"])


@pytest.fixture()
def fake_embedding():
    """384-dim fake embedding (MiniLM default)."""
    return np.random.rand(384).astype(np.float32)
