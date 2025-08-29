import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { User, UserPreferences, UserActivity } from '../types/user';
import { userService } from '../services/userService';
import { analyticsService } from '../services/analyticsService';
import { notificationService } from '../services/notificationService';

interface UserProfileProps {
  userId: number;
  onUserUpdate?: (user: User) => void;
  canEdit?: boolean;
  showActivity?: boolean;
}

interface FormData {
  name: string;
  email: string;
  bio: string;
  location: string;
  website: string;
  preferences: UserPreferences;
}

interface ValidationErrors {
  name?: string;
  email?: string;
  bio?: string;
  website?: string;
}

export const UserProfile: React.FC<UserProfileProps> = ({ 
  userId, 
  onUserUpdate, 
  canEdit = true, 
  showActivity = false 
}) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState<FormData>({
    name: '',
    email: '',
    bio: '',
    location: '',
    website: '',
    preferences: {
      emailNotifications: true,
      theme: 'light',
      language: 'en',
      timezone: 'UTC'
    }
  });
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({});
  const [userActivity, setUserActivity] = useState<UserActivity[]>([]);
  const [activityLoading, setActivityLoading] = useState(false);
  
  // Refs for cleanup and avoiding memory leaks
  const abortControllerRef = useRef<AbortController | null>(null);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Memoized validation function
  const validateForm = useMemo(() => {
    return (data: FormData): ValidationErrors => {
      const errors: ValidationErrors = {};
      
      if (!data.name.trim()) {
        errors.name = 'Name is required';
      } else if (data.name.length < 2) {
        errors.name = 'Name must be at least 2 characters';
      }
      
      // Basic email validation - could be more robust
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!data.email.trim()) {
        errors.email = 'Email is required';
      } else if (!emailRegex.test(data.email)) {
        errors.email = 'Invalid email format';
      }
      
      if (data.bio.length > 500) {
        errors.bio = 'Bio must be less than 500 characters';
      }
      
      if (data.website && !data.website.startsWith('http')) {
        errors.website = 'Website must start with http:// or https://';
      }
      
      return errors;
    };
  }, []);

  // Fetch user data with proper cleanup
  const fetchUser = useCallback(async () => {
    // Cancel previous request if still pending
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    
    abortControllerRef.current = new AbortController();
    
    try {
      setLoading(true);
      setError(null);
      
      const userData = await userService.getUser(userId, {
        signal: abortControllerRef.current.signal
      });
      
      setUser(userData);
      
      // Initialize form data - potential null pointer issues
      setFormData({
        name: userData.name || '',
        email: userData.email || '',
        bio: userData.profile?.bio || '',
        location: userData.profile?.location || '',
        website: userData.profile?.website || '',
        preferences: {
          emailNotifications: userData.preferences?.emailNotifications ?? true,
          theme: userData.preferences?.theme || 'light',
          language: userData.preferences?.language || 'en',
          timezone: userData.preferences?.timezone || 'UTC'
        }
      });
      
      // Track user profile view
      analyticsService.trackEvent('user_profile_viewed', {
        userId: userData.id,
        viewerId: getCurrentUserId() // Potential undefined
      });
      
    } catch (err: any) {
      if (err.name !== 'AbortError') {
        console.error('Failed to load user:', err);
        setError(err.message || 'Failed to load user');
        
        // Track error
        analyticsService.trackError('user_profile_load_error', {
          userId,
          error: err.message
        });
      }
    } finally {
      setLoading(false);
    }
  }, [userId]);

  // Fetch user activity
  const fetchUserActivity = useCallback(async () => {
    if (!showActivity) return;
    
    try {
      setActivityLoading(true);
      const activity = await userService.getUserActivity(userId);
      setUserActivity(activity);
    } catch (err) {
      console.error('Failed to load user activity:', err);
      // Don't show error for activity - it's supplementary data
    } finally {
      setActivityLoading(false);
    }
  }, [userId, showActivity]);

  // Effect with potential memory leak issues
  useEffect(() => {
    fetchUser();
    
    // Set up auto-refresh - potential memory leak if not cleaned up
    const refreshInterval = setInterval(() => {
      if (!isEditing) {
        fetchUser();
      }
    }, 30000); // Refresh every 30 seconds
    
    return () => {
      clearInterval(refreshInterval);
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [fetchUser, isEditing]);

  // Separate effect for activity
  useEffect(() => {
    fetchUserActivity();
  }, [fetchUserActivity]);

  // Handle form input changes
  const handleInputChange = useCallback((field: keyof FormData, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
    
    // Clear validation error for this field
    if (validationErrors[field as keyof ValidationErrors]) {
      setValidationErrors(prev => ({
        ...prev,
        [field]: undefined
      }));
    }
  }, [validationErrors]);

  // Handle save with validation and error handling
  const handleSave = useCallback(async () => {
    const errors = validateForm(formData);
    
    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      return;
    }
    
    try {
      setSaving(true);
      setError(null);
      
      const updatedUser = await userService.updateUser(userId, {
        name: formData.name,
        email: formData.email,
        profile: {
          bio: formData.bio,
          location: formData.location,
          website: formData.website
        },
        preferences: formData.preferences
      });
      
      setUser(updatedUser);
      setIsEditing(false);
      
      // Notify parent component
      onUserUpdate?.(updatedUser);
      
      // Show success notification
      notificationService.showSuccess('Profile updated successfully');
      
      // Track successful update
      analyticsService.trackEvent('user_profile_updated', {
        userId: updatedUser.id,
        fields: Object.keys(formData)
      });
      
    } catch (err: any) {
      console.error('Failed to update user:', err);
      setError(err.message || 'Failed to update user');
      
      // Track error
      analyticsService.trackError('user_profile_update_error', {
        userId,
        error: err.message
      });
      
      // Show error notification
      notificationService.showError('Failed to update profile');
    } finally {
      setSaving(false);
    }
  }, [formData, validateForm, userId, onUserUpdate]);

  // Handle edit mode toggle
  const handleEditToggle = useCallback(() => {
    if (isEditing) {
      // Reset form data when canceling
      if (user) {
        setFormData({
          name: user.name || '',
          email: user.email || '',
          bio: user.profile?.bio || '',
          location: user.profile?.location || '',
          website: user.profile?.website || '',
          preferences: user.preferences || {
            emailNotifications: true,
            theme: 'light',
            language: 'en',
            timezone: 'UTC'
          }
        });
      }
      setValidationErrors({});
    }
    setIsEditing(!isEditing);
  }, [isEditing, user]);

  // Auto-save functionality with debouncing
  useEffect(() => {
    if (isEditing && user) {
      // Clear existing timeout
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      
      // Set new timeout for auto-save
      timeoutRef.current = setTimeout(() => {
        // Auto-save draft to localStorage
        localStorage.setItem(`user-profile-draft-${userId}`, JSON.stringify(formData));
      }, 2000);
    }
    
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [formData, isEditing, userId]);

  // Load draft from localStorage on mount
  useEffect(() => {
    const savedDraft = localStorage.getItem(`user-profile-draft-${userId}`);
    if (savedDraft && isEditing) {
      try {
        const draftData = JSON.parse(savedDraft);
        setFormData(draftData);
      } catch (err) {
        console.error('Failed to load draft:', err);
      }
    }
  }, [userId, isEditing]);

  // Cleanup function
  useEffect(() => {
    return () => {
      // Clear draft when component unmounts
      localStorage.removeItem(`user-profile-draft-${userId}`);
    };
  }, [userId]);

  // Helper function - potential undefined access
  const getCurrentUserId = () => {
    // This would typically come from auth context
    return window.currentUser?.id; // Potential undefined access
  };

  // Render loading state
  if (loading) {
    return (
      <div className="user-profile loading">
        <div className="spinner" />
        <p>Loading user profile...</p>
      </div>
    );
  }

  // Render error state
  if (error) {
    return (
      <div className="user-profile error">
        <h3>Error</h3>
        <p>{error}</p>
        <button onClick={fetchUser}>Retry</button>
      </div>
    );
  }

  // Render not found state
  if (!user) {
    return (
      <div className="user-profile not-found">
        <h3>User not found</h3>
        <p>The requested user profile could not be found.</p>
      </div>
    );
  }

  return (
    <div className="user-profile">
      <div className="profile-header">
        <img 
          src={user.profile?.avatarUrl || '/default-avatar.png'} 
          alt={`${user.name}'s avatar`}
          className="avatar"
          onError={(e) => {
            // Handle broken image - potential XSS if src is user-controlled
            e.currentTarget.src = '/default-avatar.png';
          }}
        />
        <div className="header-info">
          <h2>{user.name}</h2>
          <p className="email">{user.email}</p>
          {user.profile?.location && (
            <p className="location">📍 {user.profile.location}</p>
          )}
        </div>
        {canEdit && (
          <button 
            className="edit-button"
            onClick={handleEditToggle}
            disabled={saving}
          >
            {isEditing ? 'Cancel' : 'Edit Profile'}
          </button>
        )}
      </div>

      {isEditing ? (
        <form className="edit-form" onSubmit={(e) => { e.preventDefault(); handleSave(); }}>
          <div className="form-group">
            <label htmlFor="name">Name *</label>
            <input
              id="name"
              type="text"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              className={validationErrors.name ? 'error' : ''}
              disabled={saving}
            />
            {validationErrors.name && (
              <span className="error-message">{validationErrors.name}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="email">Email *</label>
            <input
              id="email"
              type="email"
              value={formData.email}
              onChange={(e) => handleInputChange('email', e.target.value)}
              className={validationErrors.email ? 'error' : ''}
              disabled={saving}
            />
            {validationErrors.email && (
              <span className="error-message">{validationErrors.email}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="bio">Bio</label>
            <textarea
              id="bio"
              value={formData.bio}
              onChange={(e) => handleInputChange('bio', e.target.value)}
              className={validationErrors.bio ? 'error' : ''}
              disabled={saving}
              maxLength={500}
            />
            <small>{formData.bio.length}/500 characters</small>
            {validationErrors.bio && (
              <span className="error-message">{validationErrors.bio}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="website">Website</label>
            <input
              id="website"
              type="url"
              value={formData.website}
              onChange={(e) => handleInputChange('website', e.target.value)}
              className={validationErrors.website ? 'error' : ''}
              disabled={saving}
              placeholder="https://example.com"
            />
            {validationErrors.website && (
              <span className="error-message">{validationErrors.website}</span>
            )}
          </div>

          <div className="form-actions">
            <button 
              type="submit" 
              disabled={saving || Object.keys(validationErrors).length > 0}
              className="save-button"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
            <button 
              type="button" 
              onClick={handleEditToggle}
              disabled={saving}
              className="cancel-button"
            >
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <div className="profile-content">
          <div className="profile-info">
            <h3>About</h3>
            {user.profile?.bio ? (
              <p className="bio">{user.profile.bio}</p>
            ) : (
              <p className="no-bio">No bio available</p>
            )}
            
            {user.profile?.website && (
              <p className="website">
                <a 
                  href={user.profile.website} 
                  target="_blank" 
                  rel="noopener noreferrer"
                  // Potential XSS if website URL is not properly validated
                >
                  {user.profile.website}
                </a>
              </p>
            )}
          </div>

          {showActivity && (
            <div className="user-activity">
              <h3>Recent Activity</h3>
              {activityLoading ? (
                <p>Loading activity...</p>
              ) : userActivity.length > 0 ? (
                <ul className="activity-list">
                  {userActivity.map((activity, index) => (
                    <li key={index} className="activity-item">
                      <span className="activity-type">{activity.type}</span>
                      <span className="activity-date">
                        {new Date(activity.timestamp).toLocaleDateString()}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p>No recent activity</p>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};