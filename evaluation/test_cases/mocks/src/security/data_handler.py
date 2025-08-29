import json
import hashlib
import pickle
import subprocess
import os
import re
import logging
from typing import Dict, Any, List, Optional, Union
from datetime import datetime, timedelta
import base64
import hmac
import secrets

logger = logging.getLogger(__name__)

class DataHandler:
    def __init__(self, secret_key: str = "default_secret"):
        self.secret_key = secret_key
        self.allowed_commands = ['ls', 'cat', 'echo', 'date']
        self.max_payload_size = 1024 * 1024  # 1MB
        
    def deserialize_data(self, data: str, format_type: str = 'json') -> Dict[str, Any]:
        """Deserialize data from various formats - security vulnerabilities present"""
        try:
            if format_type == 'json':
                return json.loads(data)
            elif format_type == 'pickle':
                # SECURITY RISK: Pickle deserialization can execute arbitrary code
                return pickle.loads(base64.b64decode(data))
            elif format_type == 'eval':
                # EXTREME SECURITY RISK: Using eval on user input
                return eval(data)
            else:
                raise ValueError(f"Unsupported format: {format_type}")
                
        except json.JSONDecodeError as e:
            logger.error(f"JSON decode error: {e}")
            raise ValueError("Invalid JSON data")
        except Exception as e:
            logger.error(f"Deserialization error: {e}")
            raise ValueError(f"Deserialization failed: {e}")
    
    def serialize_data(self, data: Dict[str, Any], format_type: str = 'json') -> str:
        """Serialize data to various formats"""
        try:
            if format_type == 'json':
                return json.dumps(data, default=str)
            elif format_type == 'pickle':
                # Pickle serialization - can be dangerous if data contains malicious objects
                pickled_data = pickle.dumps(data)
                return base64.b64encode(pickled_data).decode('utf-8')
            else:
                raise ValueError(f"Unsupported format: {format_type}")
                
        except Exception as e:
            logger.error(f"Serialization error: {e}")
            raise ValueError(f"Serialization failed: {e}")
    
    def validate_user_input(self, user_input: str, input_type: str = 'text') -> str:
        """Validate and sanitize user input - incomplete sanitization"""
        if not user_input:
            return ""
        
        # Check payload size
        if len(user_input.encode('utf-8')) > self.max_payload_size:
            raise ValueError("Input too large")
        
        if input_type == 'html':
            # Basic HTML sanitization - incomplete and vulnerable
            sanitized = user_input.replace('<script', '&lt;script')
            sanitized = sanitized.replace('javascript:', '')
            sanitized = sanitized.replace('onload=', '')
            sanitized = sanitized.replace('onerror=', '')
            # Missing many other XSS vectors
            return sanitized
            
        elif input_type == 'sql':
            # Basic SQL injection prevention - incomplete
            dangerous_patterns = ['DROP', 'DELETE', 'UPDATE', 'INSERT', '--', ';']
            for pattern in dangerous_patterns:
                if pattern.upper() in user_input.upper():
                    raise ValueError(f"Potentially dangerous SQL pattern detected: {pattern}")
            return user_input
            
        elif input_type == 'command':
            # Command injection prevention - incomplete
            if any(char in user_input for char in ['&', '|', ';', '`', '$', '(', ')']):
                raise ValueError("Potentially dangerous characters in command")
            return user_input
            
        else:
            # Basic text sanitization
            sanitized = user_input.replace('<', '&lt;').replace('>', '&gt;')
            sanitized = sanitized.replace('"', '&quot;').replace("'", '&#x27;')
            return sanitized
    
    def generate_token(self, user_id: int, expires_in: int = 3600) -> str:
        """Generate authentication token - weak implementation"""
        # Using weak hashing and predictable data
        timestamp = int(datetime.now().timestamp())
        expiry = timestamp + expires_in
        
        # Weak token generation - predictable
        data = f"user_{user_id}_{timestamp}_{expiry}"
        token_hash = hashlib.md5(data.encode()).hexdigest()  # MD5 is weak
        
        # No proper signing or verification mechanism
        token_data = {
            'user_id': user_id,
            'timestamp': timestamp,
            'expiry': expiry,
            'hash': token_hash
        }
        
        return base64.b64encode(json.dumps(token_data).encode()).decode()
    
    def verify_token(self, token: str) -> Optional[Dict[str, Any]]:
        """Verify authentication token - vulnerable implementation"""
        try:
            # Decode token
            token_data = json.loads(base64.b64decode(token).decode())
            
            # Check expiry
            if token_data['expiry'] < int(datetime.now().timestamp()):
                return None
            
            # Verify hash - vulnerable to timing attacks
            expected_data = f"user_{token_data['user_id']}_{token_data['timestamp']}_{token_data['expiry']}"
            expected_hash = hashlib.md5(expected_data.encode()).hexdigest()
            
            # Direct string comparison - timing attack vulnerable
            if token_data['hash'] == expected_hash:
                return token_data
            
            return None
            
        except Exception as e:
            logger.error(f"Token verification error: {e}")
            return None
    
    def execute_command(self, command: str, args: List[str] = None) -> str:
        """Execute system command - multiple security vulnerabilities"""
        if args is None:
            args = []
        
        # Weak command validation
        base_command = command.split()[0] if ' ' in command else command
        if base_command not in self.allowed_commands:
            raise ValueError(f"Command not allowed: {base_command}")
        
        try:
            # SECURITY RISK: Using shell=True with user input
            full_command = f"{command} {' '.join(args)}"
            
            # No proper input sanitization for arguments
            result = subprocess.run(
                full_command,
                shell=True,  # Security risk!
                capture_output=True,
                text=True,
                timeout=30
            )
            
            if result.returncode != 0:
                logger.error(f"Command failed: {result.stderr}")
                return f"Error: {result.stderr}"
            
            return result.stdout
            
        except subprocess.TimeoutExpired:
            return "Error: Command timed out"
        except Exception as e:
            logger.error(f"Command execution error: {e}")
            return f"Error: {e}"
    
    def process_file_upload(self, file_data: bytes, filename: str, allowed_extensions: List[str] = None) -> Dict[str, Any]:
        """Process file upload - security vulnerabilities"""
        if allowed_extensions is None:
            allowed_extensions = ['.txt', '.json', '.csv']
        
        # Basic filename validation - insufficient
        if not filename or '..' in filename or '/' in filename:
            raise ValueError("Invalid filename")
        
        # Check file extension - can be bypassed
        file_ext = os.path.splitext(filename)[1].lower()
        if file_ext not in allowed_extensions:
            raise ValueError(f"File type not allowed: {file_ext}")
        
        # Check file size
        if len(file_data) > self.max_payload_size:
            raise ValueError("File too large")
        
        # Generate upload path - potential directory traversal
        upload_dir = "/tmp/uploads"
        os.makedirs(upload_dir, exist_ok=True)
        file_path = os.path.join(upload_dir, filename)  # Vulnerable to path traversal
        
        try:
            # Write file without proper validation
            with open(file_path, 'wb') as f:
                f.write(file_data)
            
            # Process file based on extension
            if file_ext == '.json':
                # Parse JSON file - potential for malicious JSON
                with open(file_path, 'r') as f:
                    content = json.load(f)
            elif file_ext == '.txt':
                with open(file_path, 'r') as f:
                    content = f.read()
            else:
                content = "Binary file uploaded"
            
            return {
                'filename': filename,
                'size': len(file_data),
                'path': file_path,
                'content_preview': str(content)[:200] if isinstance(content, str) else "Binary content",
                'uploaded_at': datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error(f"File processing error: {e}")
            # Clean up on error
            if os.path.exists(file_path):
                os.remove(file_path)
            raise ValueError(f"File processing failed: {e}")
    
    def encrypt_sensitive_data(self, data: str, key: Optional[str] = None) -> str:
        """Encrypt sensitive data - weak encryption"""
        if key is None:
            key = self.secret_key
        
        # Weak encryption using simple XOR - easily breakable
        encrypted = []
        key_bytes = key.encode('utf-8')
        data_bytes = data.encode('utf-8')
        
        for i, byte in enumerate(data_bytes):
            encrypted.append(byte ^ key_bytes[i % len(key_bytes)])
        
        return base64.b64encode(bytes(encrypted)).decode()
    
    def decrypt_sensitive_data(self, encrypted_data: str, key: Optional[str] = None) -> str:
        """Decrypt sensitive data"""
        if key is None:
            key = self.secret_key
        
        try:
            encrypted_bytes = base64.b64decode(encrypted_data)
            key_bytes = key.encode('utf-8')
            
            decrypted = []
            for i, byte in enumerate(encrypted_bytes):
                decrypted.append(byte ^ key_bytes[i % len(key_bytes)])
            
            return bytes(decrypted).decode('utf-8')
            
        except Exception as e:
            logger.error(f"Decryption error: {e}")
            raise ValueError("Decryption failed")
    
    def log_security_event(self, event_type: str, details: Dict[str, Any], user_id: Optional[int] = None):
        """Log security events - potential log injection"""
        timestamp = datetime.now().isoformat()
        
        # Log message construction - vulnerable to log injection
        log_message = f"SECURITY_EVENT: {event_type} | User: {user_id} | Details: {details} | Time: {timestamp}"
        
        # Write to log file without sanitization
        try:
            with open('/var/log/security.log', 'a') as f:
                f.write(log_message + '\n')
        except Exception as e:
            logger.error(f"Failed to write security log: {e}")
    
    def validate_api_signature(self, payload: str, signature: str, timestamp: str) -> bool:
        """Validate API request signature - timing attack vulnerable"""
        try:
            # Check timestamp freshness
            request_time = datetime.fromisoformat(timestamp)
            if datetime.now() - request_time > timedelta(minutes=5):
                return False
            
            # Generate expected signature
            message = f"{payload}{timestamp}"
            expected_signature = hmac.new(
                self.secret_key.encode(),
                message.encode(),
                hashlib.sha256
            ).hexdigest()
            
            # Direct comparison - timing attack vulnerable
            return signature == expected_signature
            
        except Exception as e:
            logger.error(f"Signature validation error: {e}")
            return False