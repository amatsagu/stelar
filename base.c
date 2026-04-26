#ifndef STELAR_BASE_H
#define STELAR_BASE_H

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <stdbool.h>

typedef enum {
    VAL_INT,
    VAL_FLOAT,
    VAL_STRING,
    VAL_PTR // Generic pointer for future structs/complex types
} ValType;

typedef struct {
    ValType type;
    union {
        int64_t i;
        double f;
        char* s;
        void* p;
    } as;
} Value;

typedef struct {
    Value* data;
    int top;
    int capacity;
    const char* name;
} Stack;

static Stack main_stack;
static Stack errors_stack;
static Stack* current_stack = &main_stack;

void stack_init(Stack* s, const char* name) {
    s->top = 0;
    s->capacity = 8; // Start small
    s->data = (Value*)malloc(sizeof(Value) * s->capacity);
    s->name = name;
}

void stelar_init() {
    stack_init(&main_stack, "main");
    stack_init(&errors_stack, "errors");
    current_stack = &main_stack;
}

static inline void stack_ensure_capacity(Stack* s) {
    if (s->top >= s->capacity) {
        s->capacity *= 2;
        s->data = (Value*)realloc(s->data, sizeof(Value) * s->capacity);
    }
}

static inline void stack_shrink_if_needed(Stack* s) {
    if (s->capacity > 8 && s->top < s->capacity / 4) {
        s->capacity /= 2;
        s->data = (Value*)realloc(s->data, sizeof(Value) * s->capacity);
    }
}

static inline Value val_int(int64_t i) { return (Value){.type = VAL_INT, .as.i = i}; }
static inline Value val_float(double f) { return (Value){.type = VAL_FLOAT, .as.f = f}; }
static inline Value val_string(char* s) { return (Value){.type = VAL_STRING, .as.s = s}; }

void push_val(Stack* s, Value v) {
    stack_ensure_capacity(s);
    s->data[s->top++] = v;
}

Value pop_val(Stack* s) {
    if (s->top <= 0) {
        fprintf(stderr, "Fatal: Stack underflow on '%s'\n", s->name);
        exit(1);
    }
    Value v = s->data[--s->top];
    stack_shrink_if_needed(s);
    return v;
}

Value peek_val(Stack* s, int index) {
    if (index < 0 || index >= s->top) {
        fprintf(stderr, "Fatal: Peek out of bounds on '%s'\n", s->name);
        exit(1);
    }
    return s->data[s->top - 1 - index];
}

int stack_size(Stack* s) { return s->top; }
void stack_reset(Stack* s) { s->top = 0; stack_shrink_if_needed(s); }

void stack_clone(Stack* s) {
    if (s->top == 0) return;
    push_val(s, s->data[s->top - 1]);
}

void stack_swap(Stack* s, int i1, int i2) {
    if (i1 < 0 || i1 >= s->top || i2 < 0 || i2 >= s->top) return;
    Value temp = s->data[s->top - 1 - i1];
    s->data[s->top - 1 - i1] = s->data[s->top - 1 - i2];
    s->data[s->top - 1 - i2] = temp;
}

void stelar_print(const char* label, Value v) {
    printf("%s", label);
    switch(v.type) {
        case VAL_INT: printf("%ld\n", v.as.i); break;
        case VAL_FLOAT: printf("%g\n", v.as.f); break;
        case VAL_STRING: printf("%s\n", v.as.s); break;
        case VAL_PTR: printf("ptr(%p)\n", v.as.p); break;
    }
}

#define STELAR_MATH(s, op, arg1_expr, arg2_expr) \
    do { \
        Value _v1 = (arg1_expr); \
        Value _v2 = (arg2_expr); \
        Value _res = {0}; \
        if (_v1.type == VAL_FLOAT || _v2.type == VAL_FLOAT) { \
            _res.type = VAL_FLOAT; \
            double a = (_v1.type == VAL_FLOAT) ? _v1.as.f : (double)_v1.as.i; \
            double b = (_v2.type == VAL_FLOAT) ? _v2.as.f : (double)_v2.as.i; \
            if ((op) == '+') _res.as.f = a + b; \
            else if ((op) == '-') _res.as.f = a - b; \
            else if ((op) == '*') _res.as.f = a * b; \
            else if ((op) == '/') _res.as.f = a / b; \
        } else { \
            _res.type = VAL_INT; \
            if ((op) == '+') _res.as.i = _v1.as.i + _v2.as.i; \
            else if ((op) == '-') _res.as.i = _v1.as.i - _v2.as.i; \
            else if ((op) == '*') _res.as.i = _v1.as.i * _v2.as.i; \
            else if ((op) == '/') _res.as.i = _v1.as.i / _v2.as.i; \
        } \
        push_val((s), _res); \
    } while(0)

#endif
