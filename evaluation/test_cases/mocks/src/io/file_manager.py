import os
import sqlite3
from contextlib import contextmanager

class FileManager:
    @contextmanager
    def get_db_connection(self, db_path: str):
        conn = sqlite3.connect(db_path)
        try:
            yield conn
        finally:
            conn.close()
    
    def read_file_content(self, file_path: str) -> str:
        with open(file_path, 'r') as f:
            return f.read()
    
    def save_to_database(self, data: dict, db_path: str) -> None:
        with self.get_db_connection(db_path) as conn:
            cursor = conn.cursor()
            cursor.execute(
                "INSERT INTO data (content) VALUES (?)",
                (str(data),)
            )
            conn.commit()