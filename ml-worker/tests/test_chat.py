"""P0 tests for ChatService (ChatServicer)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import httpx
import pytest

from ml_worker.services.chat import ChatServicer
from ml_worker.proto import chat_pb2


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _ollama_json(content: str = "Hi!", eval_count: int = 10, prompt_eval_count: int = 5) -> dict:
    return {
        "message": {"role": "assistant", "content": content},
        "eval_count": eval_count,
        "prompt_eval_count": prompt_eval_count,
    }


class _FakeHTTPResponse:
    """Minimal stand-in for httpx.Response."""

    def __init__(self, data: dict, status_code: int = 200):
        self._data = data
        self.status_code = status_code

    def json(self):
        return self._data

    def raise_for_status(self):
        if self.status_code >= 400:
            raise httpx.HTTPStatusError(
                "error", request=MagicMock(), response=MagicMock(status_code=self.status_code)
            )


# ---------------------------------------------------------------------------
# Unary Chat
# ---------------------------------------------------------------------------

class TestChatUnary:
    def setup_method(self):
        self.servicer = ChatServicer()

    @patch("ml_worker.services.chat.httpx.Client")
    def test_chat_basic(self, mock_client_cls, grpc_context, chat_request):
        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.post.return_value = _FakeHTTPResponse(_ollama_json("Hello back!"))

        resp = self.servicer.Chat(chat_request, grpc_context)

        assert resp.reply == "Hello back!"
        assert resp.tokens_used == 15
        assert resp.model == "mistral"  # default model

    @patch("ml_worker.services.chat.httpx.Client")
    def test_chat_custom_model(self, mock_client_cls, grpc_context, chat_request_full):
        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.post.return_value = _FakeHTTPResponse(_ollama_json("joke"))

        resp = self.servicer.Chat(chat_request_full, grpc_context)

        assert resp.model == "llama3"
        # Verify options were passed
        call_kwargs = mock_client.post.call_args
        body = call_kwargs[1]["json"] if "json" in call_kwargs[1] else call_kwargs.kwargs["json"]
        assert body["model"] == "llama3"
        assert "options" in body
        assert body["options"]["temperature"] == pytest.approx(0.7, abs=0.01)
        assert body["options"]["num_predict"] == 100

    @patch("ml_worker.services.chat.httpx.Client")
    def test_chat_builds_messages_with_system_and_context(self, mock_client_cls, grpc_context, chat_request_full):
        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.post.return_value = _FakeHTTPResponse(_ollama_json())

        self.servicer.Chat(chat_request_full, grpc_context)

        body = mock_client.post.call_args[1]["json"]
        messages = body["messages"]
        assert messages[0] == {"role": "system", "content": "You are helpful."}
        assert messages[1] == {"role": "user", "content": "previous context"}
        assert messages[2] == {"role": "user", "content": "Tell me a joke"}

    @patch("ml_worker.services.chat.httpx.Client")
    def test_chat_ollama_error_aborts(self, mock_client_cls, grpc_context, chat_request):
        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.post.side_effect = httpx.ConnectError("connection refused")

        with pytest.raises(Exception, match="grpc abort"):
            self.servicer.Chat(chat_request, grpc_context)


# ---------------------------------------------------------------------------
# Streaming Chat
# ---------------------------------------------------------------------------

class TestChatStream:
    def setup_method(self):
        self.servicer = ChatServicer()

    @patch("ml_worker.services.chat.httpx.Client")
    def test_stream_basic(self, mock_client_cls, grpc_context, chat_request):
        chunks_data = [
            json.dumps({"message": {"content": "Hel"}, "done": False}),
            json.dumps({"message": {"content": "lo"}, "done": False}),
            json.dumps({"message": {"content": ""}, "done": True}),
        ]

        mock_stream_resp = MagicMock()
        mock_stream_resp.raise_for_status = MagicMock()
        mock_stream_resp.iter_lines.return_value = iter(chunks_data)

        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.stream.return_value.__enter__ = MagicMock(return_value=mock_stream_resp)
        mock_client.stream.return_value.__exit__ = MagicMock(return_value=False)

        results = list(self.servicer.ChatStream(chat_request, grpc_context))

        assert len(results) == 3
        assert results[0].text == "Hel"
        assert results[0].done is False
        assert results[2].done is True

    @patch("ml_worker.services.chat.httpx.Client")
    def test_stream_error_aborts(self, mock_client_cls, grpc_context, chat_request):
        mock_client = MagicMock()
        mock_client_cls.return_value.__enter__ = MagicMock(return_value=mock_client)
        mock_client_cls.return_value.__exit__ = MagicMock(return_value=False)
        mock_client.stream.side_effect = httpx.ConnectError("fail")

        # Should call context.abort which raises
        with pytest.raises(Exception, match="grpc abort"):
            list(self.servicer.ChatStream(chat_request, grpc_context))
