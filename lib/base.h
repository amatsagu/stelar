#ifndef STELAR_BASE_H
#define STELAR_BASE_H

#if defined(__GNUC__) || defined(__clang__)
#define STELAR_INLINE static inline __attribute__((always_inline))
#else
#define STELAR_INLINE static inline
#endif

#ifndef STELAR_START_CAPACITY
#define STELAR_START_CAPACITY 32
#endif

#ifndef STELAR_MIN_SHRINK_CAPACITY
#define STELAR_MIN_SHRINK_CAPACITY 2048
#endif

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <stdbool.h>
#include <inttypes.h> // REQUIRED for cross-platform int64_t printing

typedef enum {
    VAL_INT,
    VAL_FLOAT,
    VAL_STRING,
    VAL_PTR 
} ValType; // avg. 4 bytes size

typedef struct {
    ValType type;
    union {
        int64_t i;
        double f;
        char* s;
        void* p;
    } as;
} Value; // avg. 16 bytes size

typedef struct {
    Value* data;
    size_t top;
    size_t capacity;
    const char* name;
} Stack; // avg. 32 bytes size

// Global state for a single-threaded transpiled app.
// Note: If compiling multiple C files, it should be moved to a .c file
// or use the "extern" keyword and instantiate them exactly once.
static Stack main_stack;
static Stack errors_stack;
static Stack* current_stack = &main_stack;

STELAR_INLINE void stack_init(Stack* s, const char* name) {
    s->top = 0;
    s->capacity = STELAR_START_CAPACITY;
    s->data = (Value*)malloc(sizeof(Value) * s->capacity);
    if (!s->data) {
        fprintf(stderr, "Fatal: Out of memory initializing stack '%s'\n", name);
        exit(1);
    }
    s->name = name;
}

STELAR_INLINE void stelar_init() {
    stack_init(&main_stack, "main");
    stack_init(&errors_stack, "errors");
    current_stack = &main_stack;
}

STELAR_INLINE void stack_ensure_capacity(Stack* s) {
    if (s->top >= s->capacity) {
        size_t new_cap = s->capacity * 2;
        Value* new_data = (Value*)realloc(s->data, sizeof(Value) * new_cap);
        if (!new_data) {
            fprintf(stderr, "Fatal: Out of memory resizing stack '%s'\n", s->name);
            exit(1);
        }
        s->data = new_data;
        s->capacity = new_cap;
    }
}

STELAR_INLINE void stack_shrink_if_needed(Stack* s) {
    // Only trigger if above the minimum threshold AND usage is 25% or less
    if (s->capacity >= STELAR_MIN_SHRINK_CAPACITY && s->top <= s->capacity / 4) {
        
        size_t new_cap = s->capacity / 2;
        Value* new_data = (Value*)realloc(s->data, sizeof(Value) * new_cap);
        
        if (!new_data) {
            fprintf(stderr, "Fatal: Out of memory during stack shrink on '%s'\n", s->name);
            exit(1);
        }
        
        s->data = new_data;
        s->capacity = new_cap;
    }
}

STELAR_INLINE Value val_int(int64_t i) { return (Value){.type = VAL_INT, .as.i = i}; }
STELAR_INLINE Value val_float(double f) { return (Value){.type = VAL_FLOAT, .as.f = f}; }
STELAR_INLINE Value val_string(char* s) { return (Value){.type = VAL_STRING, .as.s = s}; }

STELAR_INLINE void push_val(Stack* s, Value v) {
    stack_ensure_capacity(s);
    s->data[s->top++] = v;
}

STELAR_INLINE Value pop_val(Stack* s) {
    if (s->top == 0) {
        fprintf(stderr, "Fatal: Stack underflow on '%s'\n", s->name);
        exit(1);
    }

    stack_shrink_if_needed(s);
    return s->data[--s->top];
}

STELAR_INLINE Value peek_val(Stack* s, size_t index) {
    if (index >= s->top) {
        fprintf(stderr, "Fatal: Peek out of bounds on '%s'\n", s->name);
        exit(1);
    }
    return s->data[s->top - 1 - index];
}

STELAR_INLINE size_t stack_size(Stack* s) { return s->top; }
STELAR_INLINE void stack_reset(Stack* s) {
    s->top = 0;
    stack_shrink_if_needed(s);
}

STELAR_INLINE void stack_clone(Stack* s) {
    if (s->top == 0) return;
    push_val(s, s->data[s->top - 1]);
}

STELAR_INLINE void stack_swap(Stack* s, size_t i1, size_t i2) {
    if (i1 >= s->top || i2 >= s->top) return;
    Value temp = s->data[s->top - 1 - i1];
    s->data[s->top - 1 - i1] = s->data[s->top - 1 - i2];
    s->data[s->top - 1 - i2] = temp;
}

STELAR_INLINE void stelar_print(const char* label, Value v) {
    printf("%s", label);
    switch(v.type) {
        // PRId64 is strictly required to print int64_t correctly on Windows
        case VAL_INT: printf("%" PRId64, v.as.i); break; 
        case VAL_FLOAT: printf("%g", v.as.f); break;
        case VAL_STRING: printf("%s", v.as.s); break;
        case VAL_PTR: printf("ptr(%p)", v.as.p); break;
    }
}

STELAR_INLINE Value stelar_math(char op, Value v1, Value v2) {
    Value res = { .type = VAL_INT, .as.i = 0 }; 

    if (v1.type == VAL_FLOAT || v2.type == VAL_FLOAT) {
        double a = (v1.type == VAL_FLOAT) ? v1.as.f : (double)v1.as.i;
        double b = (v2.type == VAL_FLOAT) ? v2.as.f : (double)v2.as.i;

        if (op == '+' || op == '-' || op == '*' || op == '/') res.type = VAL_FLOAT;
        
        switch(op) {
            case '+': res.as.f = a + b; break;
            case '-': res.as.f = a - b; break;
            case '*': res.as.f = a * b; break;
            case '/': res.as.f = (b == 0.0) ? 0.0 : (a / b); break;
            case '>': res.type = VAL_INT; res.as.i = a > b; break;
            case '<': res.type = VAL_INT; res.as.i = a < b; break;
            case '=': res.type = VAL_INT; res.as.i = a == b; break;
        }
    } else {
        int64_t a = v1.as.i;
        int64_t b = v2.as.i;
        
        switch(op) {
            case '+': res.as.i = a + b; break; 
            case '-': res.as.i = a - b; break;
            case '*': res.as.i = a * b; break;
            case '/': res.as.i = (b == 0) ? 0 : (a / b); break;
            case '>': res.as.i = a > b; break;
            case '<': res.as.i = a < b; break;
            case '=': res.as.i = a == b; break;
        }
    }
    
    return res;
}

#endif