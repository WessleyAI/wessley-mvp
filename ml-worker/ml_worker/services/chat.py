"""ChatService implementation — proxies to Ollama."""

from __future__ import annotations

import json
import logging
import os
from typing import Iterator

import grpc
import httpx

from ml_worker.proto import chat_pb2, chat_pb2_grpc

logger = logging.getLogger(__name__)

_OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")
_DEFAULT_MODEL = os.getenv("CHAT_MODEL", "mistral")


class ChatServicer(chat_pb2_grpc.ChatServiceServicer):
    """Wraps Ollama /api/chat for unary and streaming chat."""

    def Chat(
        self,
        request: chat_pb2.ChatRequest,
        context: grpc.ServicerContext,
    ) -> chat_pb2.ChatResponse:
        model = request.model or _DEFAULT_MODEL
        messages = self._build_messages(request)
        options: dict = {}
        if request.temperature:
            options["temperature"] = request.temperature
        if request.max_tokens:
            options["num_predict"] = request.max_tokens

        try:
            with httpx.Client(timeout=120) as client:
                resp = client.post(
                    f"{_OLLAMA_HOST}/api/chat",
                    json={
                        "model": model,
                        "messages": messages,
                        "stream": False,
                        **({"options": options} if options else {}),
                    },
                )
                resp.raise_for_status()
                data = resp.json()
        except Exception as exc:
            logger.error("Ollama chat error: %s", exc)
            context.abort(grpc.StatusCode.INTERNAL, f"Ollama error: {exc}")
            return chat_pb2.ChatResponse()

        reply = data.get("message", {}).get("content", "")
        tokens = data.get("eval_count", 0) + data.get("prompt_eval_count", 0)
        return chat_pb2.ChatResponse(reply=reply, tokens_used=tokens, model=model)

    def ChatStream(
        self,
        request: chat_pb2.ChatRequest,
        context: grpc.ServicerContext,
    ) -> Iterator[chat_pb2.ChatChunk]:
        model = request.model or _DEFAULT_MODEL
        messages = self._build_messages(request)
        options: dict = {}
        if request.temperature:
            options["temperature"] = request.temperature

        try:
            with httpx.Client(timeout=120) as client:
                with client.stream(
                    "POST",
                    f"{_OLLAMA_HOST}/api/chat",
                    json={
                        "model": model,
                        "messages": messages,
                        "stream": True,
                        **({"options": options} if options else {}),
                    },
                ) as resp:
                    resp.raise_for_status()
                    for line in resp.iter_lines():
                        if not line:
                            continue
                        chunk = json.loads(line)
                        text = chunk.get("message", {}).get("content", "")
                        done = chunk.get("done", False)
                        yield chat_pb2.ChatChunk(text=text, done=done)
                        if done:
                            return
        except Exception as exc:
            logger.error("Ollama stream error: %s", exc)
            context.abort(grpc.StatusCode.INTERNAL, f"Ollama error: {exc}")

    @staticmethod
    def _build_messages(request: chat_pb2.ChatRequest) -> list[dict]:
        messages: list[dict] = []
        if request.system_prompt:
            messages.append({"role": "system", "content": request.system_prompt})
        for ctx in request.context:
            messages.append({"role": "user", "content": ctx})
        messages.append({"role": "user", "content": request.message})
        return messages
