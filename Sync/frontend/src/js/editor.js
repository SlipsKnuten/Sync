class Editor {
    constructor(editorElement, onUpdate, onCursorMove) {
        this.editor = editorElement;
        this.onUpdate = onUpdate;
        this.onCursorMove = onCursorMove;
        this.isRemoteUpdate = false;
        
        this.setupEventListeners();
    }

    setupEventListeners() {
        this.editor.addEventListener('input', () => {
            if (!this.isRemoteUpdate && this.onUpdate) {
                this.onUpdate(this.editor.value);
            }
        });

        // Send cursor position on various events
        ['keyup', 'keydown', 'click', 'select', 'focus'].forEach(eventType => {
            this.editor.addEventListener(eventType, () => {
                if (this.onCursorMove) {
                    this.onCursorMove(this.editor.selectionStart);
                }
            });
        });
        
        // Also send cursor position when using arrow keys or page up/down
        this.editor.addEventListener('keydown', (e) => {
            const navigationKeys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 
                                   'PageUp', 'PageDown', 'Home', 'End'];
            if (navigationKeys.includes(e.key)) {
                setTimeout(() => {
                    if (this.onCursorMove) {
                        this.onCursorMove(this.editor.selectionStart);
                    }
                }, 0);
            }
        });
    }

    updateContent(content, preserveCursor = true) {
        this.isRemoteUpdate = true;
        const cursorPos = preserveCursor ? this.editor.selectionStart : 0;
        this.editor.value = content;
        if (preserveCursor) {
            this.editor.setSelectionRange(cursorPos, cursorPos);
        }
        this.isRemoteUpdate = false;
    }

    getCursorPosition() {
        return this.editor.selectionStart;
    }

    setCursorPosition(position) {
        this.editor.setSelectionRange(position, position);
    }

    focus() {
        this.editor.focus();
    }
}