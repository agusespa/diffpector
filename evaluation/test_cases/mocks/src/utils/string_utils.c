#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void send_response(const char* response);
void log_message(const char* message);

int copy_user_input(const char* input) {
    char buffer[64];
    if (strlen(input) >= sizeof(buffer)) {
        return -1; // Input too long
    }
    strncpy(buffer, input, sizeof(buffer) - 1);
    buffer[sizeof(buffer) - 1] = '\0';
    
    printf("Processed: %s\n", buffer);
    return 0;
}

void process_network_data(char* data, size_t len) {
    char response[256];
    
    if (len >= sizeof(response)) {
        len = sizeof(response) - 1;
    }
    memcpy(response, data, len);
    response[len] = '\0';
    
    send_response(response);
}

int format_message(const char* template, const char* user_data) {
    char message[128];
    
    snprintf(message, sizeof(message), template, user_data);
    
    log_message(message);
    return 0;
}