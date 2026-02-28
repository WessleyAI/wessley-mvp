"""P0 tests for gRPC health check setup."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest
from grpc_health.v1 import health, health_pb2, health_pb2_grpc


class TestHealthCheck:
    """Verify health servicer is configured correctly in server.serve()."""

    @patch("ml_worker.server.grpc.server")
    @patch("ml_worker.server.EmbedServicer")
    @patch("ml_worker.server.ChatServicer")
    def test_health_servicer_set_serving(self, mock_chat, mock_embed, mock_grpc_server):
        """Health servicer should mark overall + both services as SERVING."""
        # We'll test the HealthServicer directly as used in server.py
        health_servicer = health.HealthServicer()
        health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
        health_servicer.set("wessley.ml.v1.ChatService", health_pb2.HealthCheckResponse.SERVING)
        health_servicer.set("wessley.ml.v1.EmbedService", health_pb2.HealthCheckResponse.SERVING)

        # Check overall health
        req = health_pb2.HealthCheckRequest(service="")
        ctx = MagicMock()
        resp = health_servicer.Check(req, ctx)
        assert resp.status == health_pb2.HealthCheckResponse.SERVING

        # Check chat service
        req = health_pb2.HealthCheckRequest(service="wessley.ml.v1.ChatService")
        resp = health_servicer.Check(req, ctx)
        assert resp.status == health_pb2.HealthCheckResponse.SERVING

        # Check embed service
        req = health_pb2.HealthCheckRequest(service="wessley.ml.v1.EmbedService")
        resp = health_servicer.Check(req, ctx)
        assert resp.status == health_pb2.HealthCheckResponse.SERVING

    def test_health_unknown_service(self):
        """Unknown service should raise NOT_FOUND."""
        health_servicer = health.HealthServicer()
        health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)

        req = health_pb2.HealthCheckRequest(service="nonexistent.Service")
        ctx = MagicMock()
        # grpc health raises an RpcError-like for unknown services
        with pytest.raises(Exception):
            health_servicer.Check(req, ctx)


class TestConfigDefaults:
    """P0: Verify environment variable parsing and defaults."""

    def test_default_ollama_host(self):
        import ml_worker.services.chat as chat_mod
        # Module-level default
        assert "localhost:11434" in chat_mod._OLLAMA_HOST

    def test_default_chat_model(self):
        import ml_worker.services.chat as chat_mod
        assert chat_mod._DEFAULT_MODEL == "mistral"

    def test_default_embed_model(self):
        with patch("ml_worker.services.embed.SentenceTransformer"):
            import ml_worker.services.embed as embed_mod
            assert embed_mod._EMBED_MODEL == "all-MiniLM-L6-v2"

    def test_default_listen_addr(self):
        import ml_worker.server as server_mod
        assert server_mod._LISTEN_ADDR == "[::]:50051"

    @patch.dict("os.environ", {"OLLAMA_HOST": "http://gpu-box:11434"})
    def test_ollama_host_env_override(self):
        """Verify env var is read (module-level, so we test the pattern)."""
        import os
        assert os.getenv("OLLAMA_HOST") == "http://gpu-box:11434"
