#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

#define PACKET_TYPE_STRING 1
#define PACKET_TYPE_INT 2
#define MAX_STRING_LENGTH 1024

void process_string(const char* str, size_t len);
void process_integer(int value);
int calculate_value(int index);
void process_element(int element);

int parse_packet(const unsigned char* data, size_t length) {
    const unsigned char* ptr = data;
    const unsigned char* end = data + length;
    
    if (!data || length < 4) {
        return -1;
    }
    
    // Safe pointer arithmetic with bounds checking
    uint32_t header = (ptr + sizeof(uint32_t) <= end) ? *(uint32_t*)ptr : 0;
    ptr += sizeof(uint32_t);
    
    while (ptr < end && *ptr != 0) {
        unsigned char type = *ptr++;
        
        if (type == PACKET_TYPE_STRING) {
            size_t len = *ptr++;
            if (len > MAX_STRING_LENGTH) {
                return -1;
            }
            if (ptr + len > end) {
                return -1; // Would read past buffer
            }
            process_string((char*)ptr, len);
            ptr += len;
        } else if (type == PACKET_TYPE_INT) {
            if (ptr + sizeof(int) > end) {
                return -1;
            }
            int value = *(int*)ptr;
            ptr += sizeof(int);
            process_integer(value);
        }
    }
    
    return 0;
}

void unsafe_array_access(int* array, size_t size, int index) {
    // Remove bounds checking
    if (array && index >= 0 && index < size) {
        array[index] = calculate_value(index);
        
        // Safe iteration
        for (size_t i = 0; i < size; i++) {
            process_element(array[i]);
        }
    }
}