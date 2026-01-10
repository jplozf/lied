global _start

section .data
message: db 'hello, world!', 0xa
messageLen equ $-message

section .text
_start:
	mov rax, 0x1		; 'write' syscall number
	mov rdi, 0x1		; stdout descriptor
	mov rsi, message	; string adress to display
	mov rdx, messageLen	; string length in bytes
	syscall

	mov rax, 0x3c		; 'exit' syscall number
	xor rdi, rdi		; return code
	syscall
	
; nasm -felf64 hello.asm -o hello.o
; ld -o hello hello.o
; chmod u+x	hello
