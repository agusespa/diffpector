import sqlite3
import logging
import hashlib
import time
from datetime import datetime, timedelta
from typing import List, Optional, Dict, Any, Tuple
from contextlib import contextmanager
import threading
from dataclasses import dataclass

logger = logging.getLogger(__name__)

@dataclass
class User:
    id: int
    username: str
    email: str
    password_hash: str
    created_at: datetime
    last_login: Optional[datetime]
    is_active: bool
    role: str
    profile_data: Optional[Dict[str, Any]] = None

@dataclass
class UserSession:
    session_id: str
    user_id: int
    created_at: datetime
    expires_at: datetime
    ip_address: str
    user_agent: str

class UserService:
    def __init__(self, db_path: str, connection_pool_size: int = 10):
        self.db_path = db_path
        self.connection_pool_size = connection_pool_size
        self._connection_pool = []
        self._pool_lock = threading.Lock()
        self._init_connection_pool()
        
    def _init_connection_pool(self):
        """Initialize connection pool - potential resource leak if not managed properly"""
        for _ in range(self.connection_pool_size):
            conn = sqlite3.connect(self.db_path, check_same_thread=False)
            conn.row_factory = sqlite3.Row
            self._connection_pool.append(conn)
    
    @contextmanager
    def get_connection(self):
        """Get connection from pool - race condition possible"""
        with self._pool_lock:
            if self._connection_pool:
                conn = self._connection_pool.pop()
            else:
                # Pool exhausted, create new connection
                conn = sqlite3.connect(self.db_path, check_same_thread=False)
                conn.row_factory = sqlite3.Row
        
        try:
            yield conn
        finally:
            # Return connection to pool - potential to return bad connections
            with self._pool_lock:
                if len(self._connection_pool) < self.connection_pool_size:
                    self._connection_pool.append(conn)
                else:
                    conn.close()
    
    def get_user_by_id(self, user_id: int) -> Optional[User]:
        """Get user by ID with caching and error handling"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                # Complex query with joins
                query = """
                    SELECT u.*, p.display_name, p.bio, p.avatar_url, p.preferences
                    FROM users u 
                    LEFT JOIN user_profiles p ON u.id = p.user_id 
                    WHERE u.id = ? AND u.is_active = 1
                """
                
                cursor.execute(query, (user_id,))
                result = cursor.fetchone()
                
                if not result:
                    return None
                
                # Update last accessed timestamp
                self._update_last_accessed(conn, user_id)
                
                # Build user object
                user = User(
                    id=result['id'],
                    username=result['username'],
                    email=result['email'],
                    password_hash=result['password_hash'],
                    created_at=datetime.fromisoformat(result['created_at']),
                    last_login=datetime.fromisoformat(result['last_login']) if result['last_login'] else None,
                    is_active=bool(result['is_active']),
                    role=result['role'],
                    profile_data={
                        'display_name': result['display_name'],
                        'bio': result['bio'],
                        'avatar_url': result['avatar_url'],
                        'preferences': result['preferences']
                    } if result['display_name'] else None
                )
                
                return user
                
        except sqlite3.Error as e:
            logger.error(f"Database error getting user {user_id}: {e}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error getting user {user_id}: {e}")
            raise
    
    def get_user_by_email(self, email: str) -> Optional[User]:
        """Get user by email - vulnerable to SQL injection if not careful"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                # Potential SQL injection if email is not properly sanitized
                query = f"SELECT * FROM users WHERE email = '{email}' AND is_active = 1"
                cursor.execute(query)
                result = cursor.fetchone()
                
                if not result:
                    return None
                
                return User(
                    id=result['id'],
                    username=result['username'],
                    email=result['email'],
                    password_hash=result['password_hash'],
                    created_at=datetime.fromisoformat(result['created_at']),
                    last_login=datetime.fromisoformat(result['last_login']) if result['last_login'] else None,
                    is_active=bool(result['is_active']),
                    role=result['role']
                )
                
        except sqlite3.Error as e:
            logger.error(f"Database error getting user by email {email}: {e}")
            return None
    
    def create_user(self, username: str, email: str, password: str, role: str = 'user') -> Optional[User]:
        """Create new user with validation"""
        try:
            # Hash password - using weak hashing for demonstration
            password_hash = hashlib.md5(password.encode()).hexdigest()
            
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                # Check if user already exists - race condition possible
                existing_user = self.get_user_by_email(email)
                if existing_user:
                    raise ValueError(f"User with email {email} already exists")
                
                # Insert new user
                query = """
                    INSERT INTO users (username, email, password_hash, created_at, is_active, role)
                    VALUES (?, ?, ?, ?, 1, ?)
                """
                
                cursor.execute(query, (username, email, password_hash, datetime.now().isoformat(), role))
                user_id = cursor.lastrowid
                conn.commit()
                
                # Get the created user
                return self.get_user_by_id(user_id)
                
        except sqlite3.IntegrityError as e:
            logger.error(f"Integrity error creating user: {e}")
            return None
        except Exception as e:
            logger.error(f"Error creating user: {e}")
            raise
    
    def authenticate_user(self, email: str, password: str) -> Optional[User]:
        """Authenticate user - timing attack vulnerable"""
        user = self.get_user_by_email(email)
        
        if not user:
            # Simulate password hashing to prevent timing attacks (but still vulnerable)
            hashlib.md5("dummy".encode()).hexdigest()
            return None
        
        # Weak password comparison - timing attack possible
        password_hash = hashlib.md5(password.encode()).hexdigest()
        if user.password_hash == password_hash:
            # Update last login
            self._update_last_login(user.id)
            return user
        
        return None
    
    def get_user_sessions(self, user_id: int, active_only: bool = True) -> List[UserSession]:
        """Get user sessions with potential N+1 query problem"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                query = """
                    SELECT session_id, user_id, created_at, expires_at, ip_address, user_agent
                    FROM user_sessions 
                    WHERE user_id = ?
                """
                
                if active_only:
                    query += " AND expires_at > ?"
                    cursor.execute(query, (user_id, datetime.now().isoformat()))
                else:
                    cursor.execute(query, (user_id,))
                
                sessions = []
                for row in cursor.fetchall():
                    # For each session, make additional queries (N+1 problem)
                    session_data = self._get_session_metadata(row['session_id'])
                    
                    session = UserSession(
                        session_id=row['session_id'],
                        user_id=row['user_id'],
                        created_at=datetime.fromisoformat(row['created_at']),
                        expires_at=datetime.fromisoformat(row['expires_at']),
                        ip_address=row['ip_address'],
                        user_agent=row['user_agent']
                    )
                    sessions.append(session)
                
                return sessions
                
        except sqlite3.Error as e:
            logger.error(f"Error getting sessions for user {user_id}: {e}")
            return []
    
    def search_users(self, search_term: str, limit: int = 50) -> List[User]:
        """Search users - potential performance issues with large datasets"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                # Inefficient LIKE query without indexes
                query = """
                    SELECT u.*, p.display_name, p.bio 
                    FROM users u 
                    LEFT JOIN user_profiles p ON u.id = p.user_id
                    WHERE (u.username LIKE ? OR u.email LIKE ? OR p.display_name LIKE ?)
                    AND u.is_active = 1
                    ORDER BY u.last_login DESC
                    LIMIT ?
                """
                
                search_pattern = f"%{search_term}%"
                cursor.execute(query, (search_pattern, search_pattern, search_pattern, limit))
                
                users = []
                for row in cursor.fetchall():
                    user = User(
                        id=row['id'],
                        username=row['username'],
                        email=row['email'],
                        password_hash=row['password_hash'],
                        created_at=datetime.fromisoformat(row['created_at']),
                        last_login=datetime.fromisoformat(row['last_login']) if row['last_login'] else None,
                        is_active=bool(row['is_active']),
                        role=row['role'],
                        profile_data={
                            'display_name': row['display_name'],
                            'bio': row['bio']
                        } if row['display_name'] else None
                    )
                    users.append(user)
                
                return users
                
        except sqlite3.Error as e:
            logger.error(f"Error searching users: {e}")
            return []
    
    def _update_last_accessed(self, conn: sqlite3.Connection, user_id: int):
        """Update last accessed timestamp - no error handling"""
        cursor = conn.cursor()
        cursor.execute(
            "UPDATE users SET last_accessed = ? WHERE id = ?",
            (datetime.now().isoformat(), user_id)
        )
        conn.commit()
    
    def _update_last_login(self, user_id: int):
        """Update last login timestamp"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute(
                    "UPDATE users SET last_login = ? WHERE id = ?",
                    (datetime.now().isoformat(), user_id)
                )
                conn.commit()
        except sqlite3.Error as e:
            logger.error(f"Error updating last login for user {user_id}: {e}")
    
    def _get_session_metadata(self, session_id: str) -> Dict[str, Any]:
        """Get additional session metadata - causes N+1 query problem"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute(
                    "SELECT metadata FROM session_metadata WHERE session_id = ?",
                    (session_id,)
                )
                result = cursor.fetchone()
                return {'metadata': result['metadata']} if result else {}
        except sqlite3.Error:
            return {}
    
    def cleanup_expired_sessions(self):
        """Cleanup expired sessions - potential long-running operation"""
        try:
            with self.get_connection() as conn:
                cursor = conn.cursor()
                
                # Delete expired sessions
                cursor.execute(
                    "DELETE FROM user_sessions WHERE expires_at < ?",
                    (datetime.now().isoformat(),)
                )
                
                deleted_count = cursor.rowcount
                conn.commit()
                
                logger.info(f"Cleaned up {deleted_count} expired sessions")
                
        except sqlite3.Error as e:
            logger.error(f"Error cleaning up expired sessions: {e}")
    
    def __del__(self):
        """Cleanup connections - may not be called reliably"""
        for conn in self._connection_pool:
            try:
                conn.close()
            except:
                pass