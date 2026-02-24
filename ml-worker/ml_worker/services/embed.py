"""EmbedService implementation — uses sentence-transformers locally."""

from __future__ import annotations

import logging
import os

import grpc
from sentence_transformers import SentenceTransformer

from ml_worker.proto import embed_pb2, embed_pb2_grpc

logger = logging.getLogger(__name__)

_EMBED_MODEL = os.getenv("EMBEDDING_MODEL_NAME", "all-MiniLM-L6-v2")


class EmbedServicer(embed_pb2_grpc.EmbedServiceServicer):
    """Local embedding via sentence-transformers."""

    def __init__(self) -> None:
        logger.info("Loading embedding model %s …", _EMBED_MODEL)
        self._model = SentenceTransformer(_EMBED_MODEL)
        logger.info("Embedding model ready (dim=%d)", self._model.get_sentence_embedding_dimension())

    def Embed(self, request: embed_pb2.EmbedRequest, context: grpc.ServicerContext) -> embed_pb2.EmbedResponse:
        vec = self._model.encode(request.text).tolist()
        return embed_pb2.EmbedResponse(values=vec, dimensions=len(vec))

    def EmbedBatch(self, request: embed_pb2.EmbedBatchRequest, context: grpc.ServicerContext) -> embed_pb2.EmbedBatchResponse:
        if not request.texts:
            return embed_pb2.EmbedBatchResponse(embeddings=[])

        vecs = self._model.encode(list(request.texts))
        embeddings = [
            embed_pb2.EmbedResponse(values=v.tolist(), dimensions=len(v))
            for v in vecs
        ]
        return embed_pb2.EmbedBatchResponse(embeddings=embeddings)
