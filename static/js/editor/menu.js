/**
 * Template Editor - Menu Module
 * Handles context menu and variable insertion
 */

function createMenuBehavior() {
    return {
        showContextMenu: false,
        contextMenuX: 0,
        contextMenuY: 0,

        getContextMenuStyle() {
            return `top: ${this.contextMenuY}px; left: ${this.contextMenuX}px`;
        },

        openContextMenu(event) {
            const menuWidth = 256;
            const menuHeight = 320;
            const padding = 10;
            
            let x = event.clientX;
            let y = event.clientY;

            if (x + menuWidth + padding > window.innerWidth) x = window.innerWidth - menuWidth - padding;
            if (y + menuHeight + padding > window.innerHeight) y = window.innerHeight - menuHeight - padding;

            this.contextMenuX = Math.max(padding, x);
            this.contextMenuY = Math.max(padding, y);
            this.showContextMenu = true;
        },

        insertVariable(tag) {
            this.insertTextAtCursor("{{ " + tag + " }}");
            this.showContextMenu = false;
        },

        insertTextAtCursor(text) {
            const sel = window.getSelection();
            if (sel.rangeCount) {
                const range = sel.getRangeAt(0);
                range.deleteContents();
                const textNode = document.createTextNode(text);
                range.insertNode(textNode);
                
                range.setStartAfter(textNode);
                range.setEndAfter(textNode);
                sel.removeAllRanges();
                sel.addRange(range);
                
                const editor = document.getElementById('editor-content');
                if (editor) editor.dispatchEvent(new Event('input', { bubbles: true }));
            }
        }
    };
}
