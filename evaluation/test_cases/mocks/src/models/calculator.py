from typing import Union, List, Optional

class Calculator:
    def add(self, a: Union[int, float], b: Union[int, float]) -> Union[int, float]:
        return a + b
    
    def divide(self, a: Union[int, float], b: Union[int, float]) -> Optional[float]:
        if b == 0:
            return None
        return a / b
    
    def calculate_average(self, numbers: List[Union[int, float]]) -> Optional[float]:
        if not numbers:
            return None
        return sum(numbers) / len(numbers)
    
    def process_data(self, data: dict) -> dict:
        return {"result": data.get("value", 0) * 2}