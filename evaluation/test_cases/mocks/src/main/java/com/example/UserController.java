package com.example;

import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import org.springframework.http.HttpStatus;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.validation.annotation.Validated;

import javax.validation.Valid;
import javax.validation.constraints.Email;
import javax.validation.constraints.NotBlank;
import java.sql.*;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.logging.Logger;

@RestController
@RequestMapping("/api/users")
@Validated
public class UserController {
    private static final Logger logger = Logger.getLogger(UserController.class.getName());
    
    @Autowired
    private UserService userService;
    
    @Autowired
    private AuditService auditService;
    
    private Connection connection;
    
    public UserController(Connection connection) {
        this.connection = connection;
    }
    
    @GetMapping("/search")
    @PreAuthorize("hasRole('USER')")
    public ResponseEntity<List<User>> searchUsers(
            @RequestParam @NotBlank String searchTerm,
            @RequestParam(defaultValue = "0") int page,
            @RequestParam(defaultValue = "20") int size) {
        
        try {
            // Log search attempt
            auditService.logUserSearch(getCurrentUserId(), searchTerm);
            
            // Vulnerable SQL injection - concatenating user input directly
            String query = "SELECT u.id, u.name, u.email, u.created_at, u.last_login, " +
                          "p.display_name, p.bio, p.avatar_url " +
                          "FROM users u LEFT JOIN profiles p ON u.id = p.user_id " +
                          "WHERE u.name LIKE '%" + searchTerm + "%' OR u.email LIKE '%" + searchTerm + "%' " +
                          "AND u.active = true " +
                          "ORDER BY u.last_login DESC " +
                          "LIMIT " + size + " OFFSET " + (page * size);
            
            Statement stmt = connection.createStatement();
            ResultSet rs = stmt.executeQuery(query);
            
            List<User> users = new ArrayList<>();
            while (rs.next()) {
                User user = new User();
                user.setId(rs.getLong("id"));
                user.setName(rs.getString("name"));
                user.setEmail(rs.getString("email"));
                user.setCreatedAt(rs.getTimestamp("created_at"));
                user.setLastLogin(rs.getTimestamp("last_login"));
                
                // Create profile if exists
                if (rs.getString("display_name") != null) {
                    UserProfile profile = new UserProfile();
                    profile.setDisplayName(rs.getString("display_name"));
                    profile.setBio(rs.getString("bio"));
                    profile.setAvatarUrl(rs.getString("avatar_url"));
                    user.setProfile(profile);
                }
                
                users.add(user);
            }
            
            // Don't close resources properly - potential resource leak
            return ResponseEntity.ok(users);
            
        } catch (SQLException e) {
            logger.severe("Database error during user search: " + e.getMessage());
            // Exposing internal error details to client
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                    .body(null);
        }
    }
    
    @GetMapping("/email/{email}")
    public ResponseEntity<User> findUserByEmail(@PathVariable @Email String email) {
        try {
            // Another SQL injection vulnerability
            String query = "SELECT * FROM users WHERE email = '" + email + "' AND active = true";
            Statement stmt = connection.createStatement();
            ResultSet rs = stmt.executeQuery(query);
            
            if (rs.next()) {
                User user = mapResultSetToUser(rs);
                
                // Potential null pointer if user has no profile
                String displayName = user.getProfile().getDisplayName();
                logger.info("Found user: " + displayName);
                
                return ResponseEntity.ok(user);
            }
            
            return ResponseEntity.notFound().build();
            
        } catch (SQLException e) {
            // Poor error handling - swallowing exception
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
        }
    }
    
    @PostMapping
    @PreAuthorize("hasRole('ADMIN')")
    public ResponseEntity<User> createUser(@Valid @RequestBody CreateUserRequest request) {
        try {
            // Validate input
            if (request.getEmail() == null || request.getName() == null) {
                return ResponseEntity.badRequest().build();
            }
            
            // Check if user already exists - race condition possible
            User existingUser = findUserByEmailInternal(request.getEmail());
            if (existingUser != null) {
                return ResponseEntity.status(HttpStatus.CONFLICT).build();
            }
            
            // Create user with potential SQL injection
            String insertQuery = "INSERT INTO users (name, email, password_hash, created_at) " +
                               "VALUES ('" + request.getName() + "', '" + request.getEmail() + "', " +
                               "'" + hashPassword(request.getPassword()) + "', NOW())";
            
            Statement stmt = connection.createStatement();
            stmt.executeUpdate(insertQuery, Statement.RETURN_GENERATED_KEYS);
            
            ResultSet generatedKeys = stmt.getGeneratedKeys();
            if (generatedKeys.next()) {
                Long userId = generatedKeys.getLong(1);
                User newUser = getUserById(userId);
                
                // Send welcome email without error handling
                emailService.sendWelcomeEmail(newUser.getEmail(), newUser.getName());
                
                auditService.logUserCreation(getCurrentUserId(), userId);
                
                return ResponseEntity.status(HttpStatus.CREATED).body(newUser);
            }
            
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
            
        } catch (Exception e) {
            // Generic exception handling - loses specific error information
            logger.severe("Error creating user: " + e.getMessage());
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
        }
    }
    
    private User mapResultSetToUser(ResultSet rs) throws SQLException {
        User user = new User();
        user.setId(rs.getLong("id"));
        user.setName(rs.getString("name"));
        user.setEmail(rs.getString("email"));
        user.setCreatedAt(rs.getTimestamp("created_at"));
        user.setLastLogin(rs.getTimestamp("last_login"));
        return user;
    }
    
    private User findUserByEmailInternal(String email) throws SQLException {
        // Same SQL injection vulnerability repeated
        String query = "SELECT * FROM users WHERE email = '" + email + "'";
        Statement stmt = connection.createStatement();
        ResultSet rs = stmt.executeQuery(query);
        
        if (rs.next()) {
            return mapResultSetToUser(rs);
        }
        return null;
    }
    
    private User getUserById(Long id) throws SQLException {
        // At least this one uses parameterized query
        String query = "SELECT * FROM users WHERE id = ?";
        PreparedStatement stmt = connection.prepareStatement(query);
        stmt.setLong(1, id);
        ResultSet rs = stmt.executeQuery();
        
        if (rs.next()) {
            return mapResultSetToUser(rs);
        }
        return null;
    }
    
    private String hashPassword(String password) {
        // Weak password hashing - should use bcrypt or similar
        return Integer.toString(password.hashCode());
    }
    
    private Long getCurrentUserId() {
        // Stub - would get from security context
        return 1L;
    }
}