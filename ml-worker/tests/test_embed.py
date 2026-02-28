"""P0 tests for EmbedService (EmbedServicer)."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import numpy as np
import pytest

from ml_worker.proto import embed_pb2


# We patch SentenceTransformer at import-time to avoid downloading a real model.
_DIM = 384


def _make_servicer():
    """Create an EmbedServicer with a mocked SentenceTransformer."""
    with patch("ml_worker.services.embed.SentenceTransformer") as mock_cls:
        mock_model = MagicMock()
        mock_model.get_sentence_embedding_dimension.return_value = _DIM
        mock_cls.return_value = mock_model

        from ml_worker.services.embed import EmbedServicer
        servicer = EmbedServicer()
    return servicer, mock_model


class TestEmbedSingle:
    def test_embed_returns_correct_dimensions(self, grpc_context, embed_request):
        servicer, mock_model = _make_servicer()
        fake_vec = np.random.rand(_DIM).astype(np.float32)
        mock_model.encode.return_value = fake_vec

        resp = servicer.Embed(embed_request, grpc_context)

        assert resp.dimensions == _DIM
        assert len(resp.values) == _DIM
        mock_model.encode.assert_called_once_with("hello world")

    def test_embed_values_match(self, grpc_context, embed_request):
        servicer, mock_model = _make_servicer()
        fake_vec = np.array([0.1, 0.2, 0.3], dtype=np.float32)
        mock_model.encode.return_value = fake_vec

        resp = servicer.Embed(embed_request, grpc_context)

        assert resp.dimensions == 3
        assert list(resp.values) == pytest.approx([0.1, 0.2, 0.3], abs=1e-5)


class TestEmbedBatch:
    def test_batch_returns_multiple(self, grpc_context, embed_batch_request):
        servicer, mock_model = _make_servicer()
        vecs = np.random.rand(2, _DIM).astype(np.float32)
        mock_model.encode.return_value = vecs

        resp = servicer.EmbedBatch(embed_batch_request, grpc_context)

        assert len(resp.embeddings) == 2
        for emb in resp.embeddings:
            assert emb.dimensions == _DIM
            assert len(emb.values) == _DIM

    def test_batch_empty_input(self, grpc_context):
        servicer, mock_model = _make_servicer()
        req = embed_pb2.EmbedBatchRequest(texts=[])

        resp = servicer.EmbedBatch(req, grpc_context)

        assert len(resp.embeddings) == 0
        mock_model.encode.assert_not_called()

    def test_batch_calls_encode_with_list(self, grpc_context, embed_batch_request):
        servicer, mock_model = _make_servicer()
        vecs = np.random.rand(2, _DIM).astype(np.float32)
        mock_model.encode.return_value = vecs

        servicer.EmbedBatch(embed_batch_request, grpc_context)

        mock_model.encode.assert_called_once_with(["hello", "world"])
