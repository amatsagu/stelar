#include <stdio.h>
#include "base.c"

void proc_main() {
    push_val(current_stack, val_int(1));
    push_val(current_stack, val_int(2));
    STELAR_MATH(
        current_stack,
        '-',
        pop_val(current_stack),
        pop_val(current_stack)
    );
    
    stelar_print("Result: ", pop_val(current_stack));
}

int main() {
    stelar_init();
    proc_main();
    return 0;
}
