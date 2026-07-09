from fastapi import FastAPI, File, Form, UploadFile, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import logging
import os
import tempfile
import time
import traceback

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("tools-ocr")

app = FastAPI(title="tools-web-ocr-service")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

OCR_MAX_EDGE = int(os.getenv("OCR_MAX_EDGE", "2048"))
OCR_LANG = os.getenv("OCR_LANG", "ch")

_engine = None


def get_engine():
    global _engine
    if _engine is None:
        try:
            from rapidocr_onnxruntime import RapidOCR

            logger.info("loading OCR engine lang=%s max_edge=%s", OCR_LANG, OCR_MAX_EDGE)
            _engine = RapidOCR()
            logger.info("OCR engine loaded")
        except Exception as exc:
            logger.error("failed to load OCR engine: %s", exc)
            raise
    return _engine


def maybe_resize(path: str) -> tuple[str, bool]:
    from PIL import Image

    with Image.open(path) as im:
        im = im.convert("RGB")
        w, h = im.size
        max_edge = max(w, h)
        if max_edge <= OCR_MAX_EDGE:
            return path, False

        scale = OCR_MAX_EDGE / max_edge
        new_size = (max(1, int(w * scale)), max(1, int(h * scale)))
        resized = im.resize(new_size, Image.Resampling.LANCZOS)
        fd, out_path = tempfile.mkstemp(suffix=".jpg")
        os.close(fd)
        resized.save(out_path, format="JPEG", quality=92)
        return out_path, True


def normalize_lang(raw: str) -> str:
    value = (raw or "ch").strip().lower()
    if value in {"ch", "zh", "cn", "chinese"}:
        return "ch"
    if value in {"en", "english"}:
        return "en"
    if value in {"ch_en", "auto", "mixed"}:
        return "ch"
    return "ch"


@app.get("/health")
def health():
    return {"status": "ok", "engine": "rapidocr", "lang": OCR_LANG}


@app.post("/v1/recognize")
async def recognize(
    file: UploadFile = File(...),
    lang: str = Form(default="ch"),
):
    suffix = os.path.splitext(file.filename or "image.png")[1] or ".png"
    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
        content = await file.read()
        tmp.write(content)
        tmp_path = tmp.name

    resized_path = tmp_path
    resized_temp = False
    started = time.perf_counter()

    try:
        resized_path, resized_temp = maybe_resize(tmp_path)
        engine = get_engine()
        _ = normalize_lang(lang)
        result, _elapsed = engine(resized_path)
        lines = []
        texts = []
        if result:
            for item in result:
                if not item or len(item) < 3:
                    continue
                box, text, score = item[0], str(item[1]).strip(), float(item[2])
                if not text:
                    continue
                lines.append(
                    {
                        "text": text,
                        "confidence": round(score, 4),
                        "box": [[round(float(p[0]), 1), round(float(p[1]), 1)] for p in box],
                    }
                )
                texts.append(text)

        duration_ms = int((time.perf_counter() - started) * 1000)
        return {
            "text": "\n".join(texts),
            "lines": lines,
            "line_count": len(lines),
            "duration_ms": duration_ms,
        }
    except Exception as exc:
        logger.error("recognize failed: %s\n%s", exc, traceback.format_exc())
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    finally:
        os.unlink(tmp_path)
        if resized_temp and os.path.exists(resized_path):
            os.unlink(resized_path)
