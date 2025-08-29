package com.example;

import java.util.List;
import java.util.ArrayList;

public class DataProcessor {
    public String processData(List<String> items) {
        StringBuilder result = new StringBuilder();
        for (String item : items) {
            result.append(item).append(" ");
        }
        return result.toString();
    }
    
    public List<String> filterItems(List<String> items, String filter) {
        List<String> filtered = new ArrayList<>();
        for (String item : items) {
            if (item.contains(filter)) {
                filtered.add(item);
            }
        }
        return filtered;
    }
}