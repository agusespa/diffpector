package com.example;

import java.util.List;
import java.util.ArrayList;

public class UserService {
    private List<User> users;
    
    public UserService() {
        // BUG: users is never initialized
    }
    
    public void addUser(User user) {
        // This will throw NullPointerException
        users.add(user);
    }
    
    public User findUserById(String id) {
        // Another potential NPE - no null check on id
        for (User user : users) {
            if (id.equals(user.getId())) {
                return user;
            }
        }
        return null;
    }
}