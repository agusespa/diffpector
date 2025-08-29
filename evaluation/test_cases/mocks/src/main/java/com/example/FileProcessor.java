package com.example;

import java.io.*;
import java.util.ArrayList;
import java.util.List;

public class FileProcessor {
    public List<String> readFile(String filename) throws IOException {
        List<String> lines = new ArrayList<>();
        BufferedReader reader = new BufferedReader(new FileReader(filename));
        
        String line;
        while ((line = reader.readLine()) != null) {
            lines.add(line);
        }
        
        reader.close();
        return lines;
    }
    
    public void writeFile(String filename, List<String> content) throws IOException {
        BufferedWriter writer = new BufferedWriter(new FileWriter(filename));
        
        for (String line : content) {
            writer.write(line);
            writer.newLine();
        }
        
        writer.close();
    }
}