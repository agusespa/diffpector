package com.example.repository;

import com.example.model.User;

public class UserRepository {
    public User findById(String id) {
        return new User(id, "Test User");
    }
    
    public int count() {
        return 100;
    }
}
