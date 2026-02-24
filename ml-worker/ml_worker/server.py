"""ML Worker gRPC server — wraps Ollama for chat and sentence-transformers for embeddings."""

from __future__ import annotations

import asyncio
import logging
import os
import signal
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

from ml_worker.services.chat import ChatServicer
from ml_worker.services.embed import EmbedServicer
from ml_worker.proto import chat_pb2, chat_pb2_grpc, embed_pb2, embed_pb2_grpc

logger = logging.getLogger(__name__)

_LISTEN_ADDR = os.getenv("LISTEN_ADDR", "[::]:50051")


def serve() -> None:
    logging.basicConfig(
        level=os.getenv("LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    # Register services
    chat_servicer = ChatServicer()
    embed_servicer = EmbedServicer()

    chat_pb2_grpc.add_ChatServiceServicer_to_server(chat_servicer, server)
    embed_pb2_grpc.add_EmbedServiceServicer_to_server(embed_servicer, server)

    # Health check
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set("wessley.ml.v1.ChatService", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set("wessley.ml.v1.EmbedService", health_pb2.HealthCheckResponse.SERVING)

    # Reflection
    service_names = (
        chat_pb2.DESCRIPTOR.services_by_name["ChatService"].full_name,
        embed_pb2.DESCRIPTOR.services_by_name["EmbedService"].full_name,
        health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(service_names, server)

    server.add_insecure_port(_LISTEN_ADDR)
    server.start()
    logger.info("ML Worker listening on %s", _LISTEN_ADDR)

    stop_event = asyncio.Event() if False else None  # noqa: kept simple

    def _stop(signum, frame):
        logger.info("Shutting down…")
        server.stop(grace=5)

    signal.signal(signal.SIGTERM, _stop)
    signal.signal(signal.SIGINT, _stop)

    server.wait_for_termination()


if __name__ == "__main__":
    serve()
