import os
import sys
import json
import time
import logging
import argparse
from pathlib import Path
from typing import Any

import pandas as pd
import numpy as np
import redis
from dotenv import load_dotenv

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class Config:
    def __init__(self):
        load_dotenv()
        self.redis_addr = os.getenv("REDIS_ADDR", "localhost:6379")
        self.redis_password = os.getenv("REDIS_PASSWORD", "")
        self.redis_db = int(os.getenv("REDIS_DB", "0"))
        self.queue_name = os.getenv("QUEUE_NAME", "metadata_jobs")
        self.db_path = os.getenv("DATABASE_PATH", "./data/dadv.db")
        self.upload_dir = os.getenv("UPLOAD_DIR", "./data/uploads")
        self.batch_size = int(os.getenv("BATCH_SIZE", "100"))


class Worker:
    def __init__(self, config: Config):
        self.config = config
        self.redis_client = redis.Redis(
            host=config.redis_addr.split(":")[0],
            port=int(config.redis_addr.split(":")[1]),
            password=config.redis_password if config.redis_password else None,
            db=config.redis_db,
            decode_responses=True
        )
        self.db = None

    def connect_db(self):
        try:
            import sqlite3
            self.db = sqlite3.connect(self.config.db_path)
            logger.info("Connected to database")
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            raise

    def process_job(self, job_data: dict) -> bool:
        job_id = job_data.get("job_id")
        file_path = job_data.get("file_path")
        file_name = job_data.get("file_name")

        logger.info(f"Processing job {job_id}: {file_name}")

        try:
            self.update_job_status(job_id, "processing")

            df = self.load_and_validate(file_path)
            if df is None:
                raise ValueError("Failed to load file")

            file_id = self.get_file_id(job_id)
            if file_id is None:
                raise ValueError("File record not found")

            records = self.process_metadata(df, file_id)
            self.insert_metadata_batch(records, file_id)

            self.analyze_and_store(df, file_id)
            self.detect_anomalies_and_pii(df, file_id)

            row_count = len(df)
            self.update_job_complete(job_id, row_count)

            # Cleanup file
            try:
                if os.path.exists(file_path):
                    os.remove(file_path)
                    logger.info(f"Cleaned up file {file_path}")
            except Exception as e:
                logger.error(f"Failed to clean up file {file_path}: {e}")

            logger.info(f"Job {job_id} completed: {row_count} records")
            return True

        except Exception as e:
            logger.error(f"Job {job_id} failed: {e}")
            self.update_job_failed(job_id, str(e))
            return False

    def load_and_validate(self, file_path: str):
        path = Path(file_path)
        if not path.exists():
            logger.error(f"File not found: {file_path}")
            return None

        ext = path.suffix.lower()

        try:
            if ext == ".csv":
                df = pd.read_csv(file_path)
            elif ext == ".json":
                df = pd.read_json(file_path)
            elif ext in [".xlsx", ".xls"]:
                df = pd.read_excel(file_path)
            else:
                logger.error(f"Unsupported file type: {ext}")
                return None

            logger.info(f"Loaded {len(df)} rows from {path.name}")
            return df

        except Exception as e:
            logger.error(f"Failed to load file: {e}")
            return None

    def process_metadata(self, df: pd.DataFrame, file_id: int):
        records = []
        columns = df.columns.tolist()

        name_col = self.find_column(columns, ["name", "filename", "file", "file_name"])
        path_col = self.find_column(columns, ["path", "filepath", "file_path", "location"])
        size_col = self.find_column(columns, ["size", "filesize", "file_size", "bytes"])
        ext_col = self.find_column(columns, ["extension", "ext", "type"])
        mime_col = self.find_column(columns, ["mime", "mime_type", "content_type"])
        created_col = self.find_column(columns, ["created", "created_at", "creation_date", "ctime"])
        modified_col = self.find_column(columns, ["modified", "modified_at", "modification_date", "mtime"])
        accessed_col = self.find_column(columns, ["accessed", "accessed_at", "access_date", "atime"])
        owner_col = self.find_column(columns, ["owner", "user", "uid", "owner_name"])
        group_col = self.find_column(columns, ["group", "gid", "group_name"])
        perms_col = self.find_column(columns, ["permissions", "perms", "mode"])

        for _, row in df.iterrows():
            record = {
                "file_id": file_id,
                "name": self.get_value(row, name_col, "unknown"),
                "path": self.get_value(row, path_col, ""),
                "size": self.get_int_value(row, size_col, 0),
                "extension": self.get_value(row, ext_col, Path(self.get_value(row, name_col, "")).suffix),
                "mime_type": self.get_value(row, mime_col, ""),
                "created_at": self.get_datetime(row, created_col),
                "modified_at": self.get_datetime(row, modified_col),
                "accessed_at": self.get_datetime(row, accessed_col),
                "owner": self.get_value(row, owner_col, ""),
                "group": self.get_value(row, group_col, ""),
                "permissions": self.get_value(row, perms_col, ""),
            }
            records.append(record)

        return records

    def find_column(self, columns: list, candidates: list):
        import difflib
        for candidate in candidates:
            for col in columns:
                if candidate.lower() in col.lower() or col.lower() in candidate.lower():
                    return col
            matches = difflib.get_close_matches(candidate, columns, n=1, cutoff=0.7)
            if matches:
                return matches[0]
        return None

    def get_value(self, row: pd.Series, col: str | None, default: str) -> str:
        if col is None or col not in row.index:
            return default
        val = row[col]
        if pd.isna(val):
            return default
        return str(val)

    def get_int_value(self, row: pd.Series, col: str | None, default: int) -> int:
        if col is None or col not in row.index:
            return default
        try:
            return int(row[col])
        except (ValueError, TypeError):
            return default

    def get_datetime(self, row: pd.Series, col: str | None) -> str:
        if col is None or col not in row.index:
            return ""
        try:
            return str(row[col])
        except Exception:
            return ""

    def insert_metadata_batch(self, records: list[dict], file_id: int):
        if not records:
            return

        cursor = self.db.cursor()
        query = """INSERT INTO metadata_records 
            (file_id, name, path, size, extension, mime_type, created_at, modified_at, accessed_at, owner, group_name, permissions)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"""

        batch_size = self.config.batch_size
        for i in range(0, len(records), batch_size):
            batch = records[i:i + batch_size]
            for r in batch:
                cursor.execute(query, (
                    file_id, r["name"], r["path"], r["size"], r["extension"],
                    r["mime_type"], r["created_at"], r["modified_at"], r["accessed_at"],
                    r["owner"], r["group"], r["permissions"]
                ))
            self.db.commit()

        logger.info(f"Inserted {len(records)} metadata records")

    def analyze_and_store(self, df: pd.DataFrame, file_id: int):
        cursor = self.db.cursor()
        
        cursor.execute("""INSERT INTO analysis_results (file_id, result_type, statistic_name, statistic_value, unit) 
            VALUES (?, ?, ?, ?, ?)""", (file_id, "count", "total_files", float(len(df)), "")
        )
        self.db.commit()

    def detect_anomalies_and_pii(self, df: pd.DataFrame, file_id: int):
        import re
        cursor = self.db.cursor()
        
        email_pattern = re.compile(r'[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+')
        phone_pattern = re.compile(r'\b\d{3}[-.]?\d{3}[-.]?\d{4}\b')
        
        for col in df.columns:
            if df[col].dtype == object:
                sample = df[col].dropna().astype(str).head(1000)
                
                if sample.apply(lambda x: bool(email_pattern.search(x))).any():
                    cursor.execute("""INSERT INTO anomalies 
                        (file_id, anomaly_type, severity, description, field, value, threshold) 
                        VALUES (?, ?, ?, ?, ?, ?, ?)""", 
                        (file_id, "PII Detected", "high", f"Potential email addresses found in column {col}", col, "N/A", 0))
                
                if sample.apply(lambda x: bool(phone_pattern.search(x))).any():
                    cursor.execute("""INSERT INTO anomalies 
                        (file_id, anomaly_type, severity, description, field, value, threshold) 
                        VALUES (?, ?, ?, ?, ?, ?, ?)""", 
                        (file_id, "PII Detected", "medium", f"Potential phone numbers found in column {col}", col, "N/A", 0))

        size_col = self.find_column(df.columns.tolist(), ["size", "filesize", "file_size", "bytes"])
        if size_col and pd.api.types.is_numeric_dtype(df[size_col]):
            mean_size = df[size_col].mean()
            std_size = df[size_col].std()
            if not pd.isna(mean_size) and not pd.isna(std_size) and std_size > 0:
                outliers = df[df[size_col] > mean_size + 3 * std_size]
                if not outliers.empty:
                    cursor.execute("""INSERT INTO anomalies 
                        (file_id, anomaly_type, severity, description, field, value, threshold) 
                        VALUES (?, ?, ?, ?, ?, ?, ?)""", 
                        (file_id, "Size Outliers", "medium", f"Found {len(outliers)} unusually large files", size_col, f"Max: {outliers[size_col].max()}", mean_size + 3 * std_size))

        self.db.commit()

    def get_file_id(self, job_id: str):
        cursor = self.db.cursor()
        cursor.execute("SELECT id FROM files WHERE job_id = ?", (job_id,))
        result = cursor.fetchone()
        return result[0] if result else None

    def update_job_status(self, job_id: str, status: str):
        job_key = f"job:{job_id}"
        data = {"status": status, "updated": time.time()}
        self.redis_client.set(job_key, json.dumps(data), ex=86400)

    def update_job_complete(self, job_id: str, row_count: int):
        job_key = f"job:{job_id}"
        data = {"status": "completed", "row_count": row_count, "completed": time.time()}
        self.redis_client.set(job_key, json.dumps(data), ex=86400)

        cursor = self.db.cursor()
        cursor.execute("UPDATE files SET status = 'completed' WHERE job_id = ?", (job_id,))
        self.db.commit()

    def update_job_failed(self, job_id: str, error: str):
        job_key = f"job:{job_id}"
        data = {"status": "failed", "error": error, "completed": time.time()}
        self.redis_client.set(job_key, json.dumps(data), ex=86400)

        cursor = self.db.cursor()
        cursor.execute("UPDATE files SET status = 'failed' WHERE job_id = ?", (job_id,))
        self.db.commit()

    def run(self):
        self.connect_db()

        logger.info(f"Worker started, waiting for jobs on queue '{self.config.queue_name}'")

        while True:
            try:
                result = self.redis_client.blpop(self.config.queue_name, timeout=60)
                if result is None:
                    continue

                _, job_json = result
                job_data = json.loads(job_json)
                self.process_job(job_data)

            except KeyboardInterrupt:
                logger.info("Worker shutting down")
                break
            except Exception as e:
                logger.error(f"Worker error: {e}")
                time.sleep(5)


def main():
    parser = argparse.ArgumentParser(description="DADV Metadata Worker")
    parser.add_argument("--config", default=".env", help="Config file")
    args = parser.parse_args()

    config = Config()
    worker = Worker(config)
    worker.run()


if __name__ == "__main__":
    main()