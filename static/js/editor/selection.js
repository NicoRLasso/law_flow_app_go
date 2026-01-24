/**
 * Template Editor - Selection Module
 * Handles saving and restoring editor selection for toolbar operations
 */

function createSelectionBehavior() {
    return {
        // Track the last valid editor selection for toolbar operations
        lastEditorSelection: null,

        // Track selection when it's inside the editor
        trackEditorSelection() {
            const editor = document.getElementById('editor-content');
            const selection = window.getSelection();
            if (!editor || !selection.rangeCount) return;

            let node = selection.getRangeAt(0).commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;

            // Only save if selection is within the editor
            if (editor.contains(node)) {
                this.lastEditorSelection = this.saveSelection();
            }
        },

        // Save selection before opening toolbars
        saveEditorSelection() {
            const editor = document.getElementById('editor-content');
            const selection = window.getSelection();
            if (!editor || !selection.rangeCount) return;

            const range = selection.getRangeAt(0);
            let node = range.commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;

            if (editor.contains(node)) {
                this.lastEditorSelection = {
                    startContainer: range.startContainer,
                    startOffset: range.startOffset,
                    endContainer: range.endContainer,
                    endOffset: range.endOffset
                };
            }
        },

        // Restore the saved editor selection and focus the editor
        restoreEditorSelection() {
            const editor = document.getElementById('editor-content');
            if (!editor || !this.lastEditorSelection) return false;

            // Check if nodes are still in DOM
            if (!document.contains(this.lastEditorSelection.startContainer) ||
                !document.contains(this.lastEditorSelection.endContainer)) {
                return false;
            }

            try {
                editor.focus();
                const range = document.createRange();
                range.setStart(this.lastEditorSelection.startContainer, this.lastEditorSelection.startOffset);
                range.setEnd(this.lastEditorSelection.endContainer, this.lastEditorSelection.endOffset);
                const sel = window.getSelection();
                sel.removeAllRanges();
                sel.addRange(range);
                return true;
            } catch (e) {
                console.warn('Could not restore editor selection:', e);
                return false;
            }
        },

        // Generic save selection utility
        saveSelection() {
            const sel = window.getSelection();
            if (!sel.rangeCount) return null;
            const range = sel.getRangeAt(0);
            return {
                startContainer: range.startContainer,
                startOffset: range.startOffset,
                endContainer: range.endContainer,
                endOffset: range.endOffset
            };
        },

        // Generic restore selection utility
        restoreSelection(saved) {
            if (!saved) return;
            
            // Safety: check if nodes remain connected
            if (!document.contains(saved.startContainer) || !document.contains(saved.endContainer)) {
                return;
            }

            try {
                const range = document.createRange();
                range.setStart(saved.startContainer, saved.startOffset);
                range.setEnd(saved.endContainer, saved.endOffset);
                
                const sel = window.getSelection();
                sel.removeAllRanges();
                sel.addRange(range);
            } catch (e) {
                console.warn('Could not restore selection safely:', e);
            }
        }
    };
}
