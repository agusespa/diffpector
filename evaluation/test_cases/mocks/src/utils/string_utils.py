from typing import List, Optional

class StringUtils:
    def capitalize_words(self, text: str) -> str:
        return ' '.join(word.capitalize() for word in text.split())
    
    def find_longest_word(self, words: List[str]) -> Optional[str]:
        if not words:
            return None
        longest = words[0]
        for word in words[1:]:
            if len(word) > len(longest):
                longest = word
        return longest
    
    def remove_duplicates(self, items: List[str]) -> List[str]:
        return list(set(items))