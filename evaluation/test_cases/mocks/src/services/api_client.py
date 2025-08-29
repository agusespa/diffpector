import asyncio
import aiohttp
import logging
import time
from typing import List, Dict, Any, Optional, Tuple
from dataclasses import dataclass
from datetime import datetime, timedelta
import json
import ssl
import certifi

logger = logging.getLogger(__name__)

@dataclass
class ApiResponse:
    status_code: int
    data: Dict[str, Any]
    headers: Dict[str, str]
    response_time: float
    url: str

@dataclass
class RateLimitInfo:
    requests_remaining: int
    reset_time: datetime
    limit_per_hour: int

class ApiClient:
    def __init__(self, base_url: str, api_key: str, timeout: int = 30, max_retries: int = 3):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self.max_retries = max_retries
        self.session: Optional[aiohttp.ClientSession] = None
        self.rate_limit_info: Optional[RateLimitInfo] = None
        self._request_count = 0
        self._last_request_time = 0
        
    async def __aenter__(self):
        await self._ensure_session()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def _ensure_session(self):
        """Ensure session is created with proper SSL context"""
        if not self.session or self.session.closed:
            # Create SSL context - potential security issue with verification disabled
            ssl_context = ssl.create_default_context(cafile=certifi.where())
            ssl_context.check_hostname = False  # Security risk!
            ssl_context.verify_mode = ssl.CERT_NONE  # Security risk!
            
            connector = aiohttp.TCPConnector(
                ssl=ssl_context,
                limit=100,
                limit_per_host=30,
                keepalive_timeout=30,
                enable_cleanup_closed=True
            )
            
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            
            self.session = aiohttp.ClientSession(
                connector=connector,
                timeout=timeout,
                headers={
                    'Authorization': f'Bearer {self.api_key}',
                    'User-Agent': 'ApiClient/1.0',
                    'Accept': 'application/json'
                }
            )
    
    async def fetch_data(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> ApiResponse:
        """Fetch data from API endpoint with retry logic"""
        url = f"{self.base_url}/{endpoint.lstrip('/')}"
        
        await self._ensure_session()
        
        # Rate limiting check
        await self._check_rate_limit()
        
        for attempt in range(self.max_retries + 1):
            try:
                start_time = time.time()
                
                async with self.session.get(url, params=params) as response:
                    response_time = time.time() - start_time
                    
                    # Update rate limit info from headers
                    self._update_rate_limit_info(response.headers)
                    
                    # Handle different status codes
                    if response.status == 429:  # Rate limited
                        retry_after = int(response.headers.get('Retry-After', 60))
                        logger.warning(f"Rate limited, waiting {retry_after} seconds")
                        await asyncio.sleep(retry_after)
                        continue
                    
                    if response.status >= 500:  # Server error
                        if attempt < self.max_retries:
                            wait_time = 2 ** attempt  # Exponential backoff
                            logger.warning(f"Server error {response.status}, retrying in {wait_time}s")
                            await asyncio.sleep(wait_time)
                            continue
                        else:
                            response.raise_for_status()
                    
                    if response.status >= 400:
                        logger.error(f"Client error {response.status} for {url}")
                        response.raise_for_status()
                    
                    # Parse response
                    try:
                        data = await response.json()
                    except json.JSONDecodeError:
                        # Fallback to text if JSON parsing fails
                        text_data = await response.text()
                        data = {'raw_response': text_data}
                    
                    return ApiResponse(
                        status_code=response.status,
                        data=data,
                        headers=dict(response.headers),
                        response_time=response_time,
                        url=url
                    )
                    
            except asyncio.TimeoutError:
                if attempt < self.max_retries:
                    wait_time = 2 ** attempt
                    logger.warning(f"Timeout for {url}, retrying in {wait_time}s")
                    await asyncio.sleep(wait_time)
                    continue
                else:
                    logger.error(f"Final timeout for {url}")
                    raise
                    
            except aiohttp.ClientError as e:
                if attempt < self.max_retries:
                    wait_time = 2 ** attempt
                    logger.warning(f"Client error for {url}: {e}, retrying in {wait_time}s")
                    await asyncio.sleep(wait_time)
                    continue
                else:
                    logger.error(f"Final client error for {url}: {e}")
                    raise
        
        raise Exception(f"Max retries exceeded for {url}")
    
    async def fetch_multiple(self, endpoints: List[str], params_list: Optional[List[Dict[str, Any]]] = None) -> List[ApiResponse]:
        """Fetch multiple endpoints concurrently - potential resource exhaustion"""
        if params_list is None:
            params_list = [None] * len(endpoints)
        
        # Create all tasks at once - could overwhelm the server
        tasks = [
            self.fetch_data(endpoint, params) 
            for endpoint, params in zip(endpoints, params_list)
        ]
        
        # Execute all requests concurrently without limiting concurrency
        try:
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Process results and handle exceptions
            api_responses = []
            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    logger.error(f"Error fetching {endpoints[i]}: {result}")
                    # Create error response
                    api_responses.append(ApiResponse(
                        status_code=500,
                        data={'error': str(result)},
                        headers={},
                        response_time=0.0,
                        url=endpoints[i]
                    ))
                else:
                    api_responses.append(result)
            
            return api_responses
            
        except Exception as e:
            logger.error(f"Error in batch fetch: {e}")
            raise
    
    async def process_async_data(self, data_sources: List[str]) -> Dict[str, Any]:
        """Process data from multiple sources with potential async issues"""
        results = {}
        errors = []
        
        try:
            # Fetch all data sources
            api_responses = await self.fetch_multiple(data_sources)
            
            # Process each response
            processing_tasks = []
            for i, response in enumerate(api_responses):
                if response.status_code == 200:
                    # Create async task for processing - potential race conditions
                    task = asyncio.create_task(
                        self._process_single_response(data_sources[i], response.data)
                    )
                    processing_tasks.append((data_sources[i], task))
                else:
                    errors.append(f"Failed to fetch {data_sources[i]}: {response.status_code}")
            
            # Wait for all processing to complete
            for source, task in processing_tasks:
                try:
                    processed_data = await task
                    results[source] = processed_data
                    
                    # Save result asynchronously - fire and forget (potential data loss)
                    asyncio.create_task(self.save_result(source, processed_data))
                    
                except Exception as e:
                    logger.error(f"Error processing {source}: {e}")
                    errors.append(f"Processing error for {source}: {e}")
            
            # Update metrics
            await self._update_processing_metrics(len(results), len(errors))
            
            return {
                'results': results,
                'errors': errors,
                'total_processed': len(results),
                'total_errors': len(errors)
            }
            
        except Exception as e:
            logger.error(f"Critical error in process_async_data: {e}")
            raise
    
    async def _process_single_response(self, source: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """Process individual response data"""
        # Simulate complex processing
        await asyncio.sleep(0.1)
        
        # Transform data based on source
        if 'users' in source:
            return self._transform_user_data(data)
        elif 'orders' in source:
            return self._transform_order_data(data)
        else:
            return self._transform_generic_data(data)
    
    def _transform_user_data(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Transform user data - potential KeyError if structure is unexpected"""
        return {
            'user_count': len(data['users']),  # KeyError if 'users' key missing
            'active_users': len([u for u in data['users'] if u['active']]),  # KeyError if 'active' missing
            'processed_at': datetime.now().isoformat()
        }
    
    def _transform_order_data(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Transform order data"""
        orders = data.get('orders', [])
        total_value = sum(order.get('total', 0) for order in orders)
        
        return {
            'order_count': len(orders),
            'total_value': total_value,
            'average_order_value': total_value / len(orders) if orders else 0,  # Division by zero risk
            'processed_at': datetime.now().isoformat()
        }
    
    def _transform_generic_data(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Transform generic data"""
        return {
            'record_count': len(data) if isinstance(data, (list, dict)) else 1,
            'data_type': type(data).__name__,
            'processed_at': datetime.now().isoformat()
        }
    
    async def save_result(self, source: str, result: Dict[str, Any]) -> None:
        """Save processing result - potential data loss if fails"""
        try:
            # Simulate database save with potential connection issues
            await asyncio.sleep(0.05)
            
            # Log successful save
            logger.info(f"Saved result for {source}: {len(result)} records")
            
        except Exception as e:
            # Swallow exceptions - data loss risk
            logger.error(f"Failed to save result for {source}: {e}")
    
    async def _check_rate_limit(self):
        """Check and enforce rate limiting"""
        if self.rate_limit_info:
            if self.rate_limit_info.requests_remaining <= 0:
                wait_time = (self.rate_limit_info.reset_time - datetime.now()).total_seconds()
                if wait_time > 0:
                    logger.info(f"Rate limit reached, waiting {wait_time:.2f} seconds")
                    await asyncio.sleep(wait_time)
        
        # Simple rate limiting - not thread-safe
        current_time = time.time()
        if current_time - self._last_request_time < 0.1:  # 10 requests per second max
            await asyncio.sleep(0.1)
        
        self._last_request_time = current_time
        self._request_count += 1
    
    def _update_rate_limit_info(self, headers: Dict[str, str]):
        """Update rate limit information from response headers"""
        try:
            if 'X-RateLimit-Remaining' in headers:
                self.rate_limit_info = RateLimitInfo(
                    requests_remaining=int(headers['X-RateLimit-Remaining']),
                    reset_time=datetime.fromtimestamp(int(headers.get('X-RateLimit-Reset', 0))),
                    limit_per_hour=int(headers.get('X-RateLimit-Limit', 1000))
                )
        except (ValueError, KeyError) as e:
            logger.warning(f"Failed to parse rate limit headers: {e}")
    
    async def _update_processing_metrics(self, success_count: int, error_count: int):
        """Update processing metrics"""
        # Simulate metrics update
        await asyncio.sleep(0.01)
        logger.info(f"Processing metrics: {success_count} success, {error_count} errors")
    
    async def close(self):
        """Properly close the session"""
        if self.session and not self.session.closed:
            await self.session.close()
    
    def __del__(self):
        """Cleanup - may not be called reliably"""
        if self.session and not self.session.closed:
            # This is problematic - can't use await in __del__
            try:
                loop = asyncio.get_event_loop()
                if loop.is_running():
                    loop.create_task(self.session.close())
            except:
                pass