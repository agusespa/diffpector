from typing import List, Set

class DataProcessor:
    def find_duplicates(self, items: List[str]) -> Set[str]:
        seen = set()
        duplicates = set()
        for item in items:
            if item in seen:
                duplicates.add(item)
            else:
                seen.add(item)
        return duplicates
    
    def process_large_dataset(self, data: List[dict]) -> List[dict]:
        return [item for item in data if item.get('active', False)]