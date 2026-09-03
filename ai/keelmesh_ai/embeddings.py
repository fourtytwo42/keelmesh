from __future__ import annotations

import threading
from pathlib import Path

import numpy as np
import onnxruntime as ort
from tokenizers import Tokenizer


MODEL_VERSION = "all-MiniLM-L6-v2-onnx-v1"


class MiniLMEmbedder:
    """Pinned, CPU-only sentence embeddings with attention-mask mean pooling."""

    def __init__(self, model_dir: str = "/models/all-MiniLM-L6-v2") -> None:
        root = Path(model_dir)
        self._tokenizer = Tokenizer.from_file(str(root / "tokenizer.json"))
        self._tokenizer.enable_truncation(max_length=256)
        self._tokenizer.enable_padding(length=None, pad_id=0, pad_token="[PAD]")
        options = ort.SessionOptions()
        options.intra_op_num_threads = 2
        options.inter_op_num_threads = 1
        self._session = ort.InferenceSession(
            str(root / "model.onnx"), options, providers=["CPUExecutionProvider"]
        )
        self._lock = threading.Lock()

    def encode(self, texts: list[str]) -> list[list[float]]:
        if not texts or len(texts) > 32:
            raise ValueError("embed request must contain 1-32 texts")
        if any(not value.strip() or len(value) > 24_000 for value in texts):
            raise ValueError("embed text must contain 1-24000 characters")
        encoded = self._tokenizer.encode_batch(texts)
        ids = np.asarray([item.ids for item in encoded], dtype=np.int64)
        masks = np.asarray([item.attention_mask for item in encoded], dtype=np.int64)
        types = np.asarray([item.type_ids for item in encoded], dtype=np.int64)
        inputs: dict[str, np.ndarray] = {"input_ids": ids, "attention_mask": masks}
        if "token_type_ids" in {item.name for item in self._session.get_inputs()}:
            inputs["token_type_ids"] = types
        with self._lock:
            token_embeddings = self._session.run(None, inputs)[0]
        expanded = np.expand_dims(masks, -1).astype(np.float32)
        pooled = (token_embeddings * expanded).sum(axis=1) / np.clip(expanded.sum(axis=1), 1e-9, None)
        pooled /= np.clip(np.linalg.norm(pooled, axis=1, keepdims=True), 1e-12, None)
        if pooled.shape[1] != 384:
            raise RuntimeError(f"unexpected embedding width {pooled.shape[1]}")
        return pooled.astype(np.float32).tolist()
