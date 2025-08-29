package com.example;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.cache.annotation.CacheEvict;

import java.util.List;
import java.util.ArrayList;
import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.logging.Logger;
import java.time.LocalDateTime;
import java.util.stream.Collectors;

@Service
@Transactional
public class UserService {
    private static final Logger logger = Logger.getLogger(UserService.class.getName());
    
    @Autowired
    private UserRepository userRepository;
    
    @Autowired
    private EmailService emailService;
    
    @Autowired
    private AuditService auditService;
    
    // Potential resource leak - executor not properly managed
    private ExecutorService executorService = Executors.newFixedThreadPool(10);
    
    // BUG: users list is never initialized but used in methods
    private List<User> users;
    
    public UserService() {
        // BUG: users is never initialized, will cause NullPointerException
    }
    
    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
        // Still not initializing users list
    }
    
    @Cacheable("users")
    public User getUserById(Long id) {
        // No null check on id parameter
        logger.info("Fetching user with id: " + id);
        
        // Potential NPE if userRepository is null
        User user = userRepository.findById(id);
        
        if (user != null) {
            // Update last accessed timestamp asynchronously
            CompletableFuture.runAsync(() -> {
                try {
                    user.setLastAccessed(LocalDateTime.now());
                    userRepository.save(user);
                } catch (Exception e) {
                    // Swallowing exception - potential data inconsistency
                    logger.severe("Failed to update last accessed: " + e.getMessage());
                }
            }, executorService);
        }
        
        return user;
    }
    
    public List<User> getAllUsers() {
        try {
            // Potential performance issue - loading all users without pagination
            List<User> allUsers = userRepository.findAll();
            
            // N+1 query problem - loading additional data for each user
            for (User user : allUsers) {
                user.setProfile(userRepository.findProfileByUserId(user.getId()));
                user.setPreferences(userRepository.findPreferencesByUserId(user.getId()));
            }
            
            return allUsers;
            
        } catch (Exception e) {
            logger.severe("Error fetching all users: " + e.getMessage());
            // Returning null instead of empty list - potential NPE for callers
            return null;
        }
    }
    
    public User findUserById(String id) {
        // Another potential NPE - no null check on id parameter
        // Also using String id instead of Long - inconsistent with getUserById
        for (User user : users) {  // NPE here - users is never initialized
            if (id.equals(user.getId().toString())) {
                return user;
            }
        }
        return null;
    }
    
    public void addUser(User user) {
        // This will throw NullPointerException because users is never initialized
        users.add(user);
        
        // Also save to repository
        try {
            userRepository.save(user);
            auditService.logUserCreation(user.getId(), getCurrentUserId());
        } catch (Exception e) {
            // Remove from list if database save fails - but list operation already failed
            users.remove(user);
            throw new RuntimeException("Failed to save user", e);
        }
    }
    
    @CacheEvict(value = "users", key = "#user.id")
    public User createUser(String name, String email) {
        // Basic validation - insufficient
        if (name == null || email == null) {
            throw new IllegalArgumentException("Name and email are required");
        }
        
        // Check for duplicate email - race condition possible
        User existingUser = userRepository.findByEmail(email);
        if (existingUser != null) {
            throw new IllegalStateException("User with email already exists: " + email);
        }
        
        User user = new User(name, email);
        user.setCreatedAt(LocalDateTime.now());
        user.setActive(true);
        
        try {
            // Save user
            User savedUser = userRepository.save(user);
            
            // Send welcome email asynchronously - fire and forget (potential failure)
            CompletableFuture.runAsync(() -> {
                try {
                    emailService.sendWelcomeEmail(savedUser.getEmail(), savedUser.getName());
                } catch (Exception e) {
                    // Email failure doesn't affect user creation - but no retry mechanism
                    logger.warning("Failed to send welcome email: " + e.getMessage());
                }
            }, executorService);
            
            // Add to in-memory list - will cause NPE
            users.add(savedUser);
            
            return savedUser;
            
        } catch (Exception e) {
            logger.severe("Error creating user: " + e.getMessage());
            throw new RuntimeException("User creation failed", e);
        }
    }
    
    public List<User> searchUsers(String searchTerm, int page, int size) {
        if (searchTerm == null || searchTerm.trim().isEmpty()) {
            return new ArrayList<>();
        }
        
        try {
            // Inefficient search - no indexing, case-sensitive
            List<User> allUsers = userRepository.findAll();
            
            return allUsers.stream()
                    .filter(user -> user.getName().contains(searchTerm) || 
                                  user.getEmail().contains(searchTerm))
                    .skip(page * size)
                    .limit(size)
                    .collect(Collectors.toList());
                    
        } catch (Exception e) {
            logger.severe("Error searching users: " + e.getMessage());
            return new ArrayList<>();
        }
    }
    
    public void deleteUser(Long userId) {
        // No validation of userId
        User user = getUserById(userId);
        
        if (user == null) {
            throw new IllegalArgumentException("User not found: " + userId);
        }
        
        try {
            // Soft delete
            user.setActive(false);
            user.setDeletedAt(LocalDateTime.now());
            userRepository.save(user);
            
            // Remove from in-memory list - NPE risk
            users.remove(user);
            
            // Audit log
            auditService.logUserDeletion(userId, getCurrentUserId());
            
        } catch (Exception e) {
            logger.severe("Error deleting user: " + e.getMessage());
            throw new RuntimeException("User deletion failed", e);
        }
    }
    
    public CompletableFuture<List<User>> getUsersAsync(List<Long> userIds) {
        // Async method with potential issues
        return CompletableFuture.supplyAsync(() -> {
            List<User> result = new ArrayList<>();
            
            // Sequential processing instead of parallel - inefficient
            for (Long id : userIds) {
                try {
                    User user = getUserById(id);
                    if (user != null) {
                        result.add(user);
                    }
                } catch (Exception e) {
                    // Continue processing other users - partial failure handling
                    logger.warning("Failed to fetch user " + id + ": " + e.getMessage());
                }
            }
            
            return result;
        }, executorService);
    }
    
    public void updateUserLastLogin(Long userId) {
        // Method with potential concurrency issues
        User user = getUserById(userId);
        if (user != null) {
            user.setLastLogin(LocalDateTime.now());
            user.setLoginCount(user.getLoginCount() + 1);  // Race condition possible
            
            // Save without proper error handling
            userRepository.save(user);
        }
    }
    
    private Long getCurrentUserId() {
        // Stub method - would get from security context
        return 1L;
    }
    
    // Missing proper cleanup method for executor service
    // This could lead to resource leaks when the service is destroyed
}