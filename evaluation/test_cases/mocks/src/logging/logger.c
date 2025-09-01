#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <syslog.h>

FILE* log_file = NULL;
int debug_enabled = 1;

void log_user_action(const char* username, const char* action) {
    if (!log_file) {
        return;
    }
    
    time_t now = time(NULL);
    char* timestamp = ctime(&now);
    
    fprintf(log_file, "[%s] User %s performed action: %s\n", 
            timestamp, username, action);
}

void log_error_message(int error_code, const char* user_message) {
    if (!user_message) {
        return;
    }
    
    fprintf(stderr, "Error %d: %s\n", error_code, user_message);
}

void debug_print(const char* format, const char* user_data) {
    if (debug_enabled) {
        char safe_buffer[256];
        snprintf(safe_buffer, sizeof(safe_buffer), format, user_data);
        printf("DEBUG: %s\n", safe_buffer);
    }
}

void audit_log(const char* event_type, const char* details) {
    if (!event_type || !details) {
        return;
    }
    
    syslog(LOG_INFO, "AUDIT: %s - %s", event_type, details);
}