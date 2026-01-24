/**
 * Template Editor - Formatting Module
 * Handles text formatting: bold, italic, underline, alignment, headings, font size, colors
 */

function createFormattingBehavior() {
    return {
        // Formatting state
        isBold: false,
        isItalic: false,
        isUnderline: false,
        alignment: 'left',
        currentHeading: 'Normal',
        currentFontSize: 12,
        currentTextColor: '#000000',
        customColor: '#000000',

        /**
         * Execute a basic formatting command (bold, italic, underline, alignment)
         */
        execCommand(command) {
            this.restoreEditorSelection();
            document.execCommand(command, false, null);
            this.updateFormattingState();
            this.trackEditorSelection();
        },

        /**
         * Set the heading level (h1, h2, h3, h4, p)
         */
        setHeading(tag) {
            const editor = document.getElementById('editor-content');

            // Try to restore saved selection
            this.restoreEditorSelection();

            const selection = window.getSelection();
            if (!selection.rangeCount) {
                if (editor) editor.focus();
                console.warn('No selection to apply heading to');
                return;
            }

            // Check if selection is within editor
            const range = selection.getRangeAt(0);
            let node = range.commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;
            if (editor && !editor.contains(node)) {
                console.warn('Selection is not within editor');
                return;
            }

            // Try different formats for browser compatibility
            let success = document.execCommand('formatBlock', false, tag);
            if (!success) {
                success = document.execCommand('formatBlock', false, '<' + tag + '>');
            }

            this.updateFormattingState();
            this.trackEditorSelection();

            if (editor) editor.dispatchEvent(new Event('input', { bubbles: true }));
        },

        /**
         * Set the font size in points
         */
        setFontSize(size) {
            const editor = document.getElementById('editor-content');

            // Try to restore saved selection, but continue even if it fails
            this.restoreEditorSelection();

            const selection = window.getSelection();
            if (!selection.rangeCount) {
                // If no selection, focus editor and place cursor at end
                if (editor) {
                    editor.focus();
                }
                console.warn('No selection to apply font size to');
                return;
            }

            const range = selection.getRangeAt(0);

            // Check if selection is within editor
            let node = range.commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;
            if (editor && !editor.contains(node)) {
                console.warn('Selection is not within editor');
                return;
            }

            // If no selection, create a span for future typing
            if (range.collapsed) {
                const span = document.createElement('span');
                span.style.fontSize = size + 'pt';
                span.innerHTML = '&#8203;'; // Zero-width space
                range.insertNode(span);

                range.setStart(span, 1);
                range.setEnd(span, 1);
                selection.removeAllRanges();
                selection.addRange(range);
            } else {
                // Wrap selected content in a span with the font size
                const contents = range.extractContents();
                const span = document.createElement('span');
                span.style.fontSize = size + 'pt';
                span.appendChild(contents);
                range.insertNode(span);

                range.selectNodeContents(span);
                selection.removeAllRanges();
                selection.addRange(range);
            }

            this.currentFontSize = size;
            this.trackEditorSelection();

            if (editor) editor.dispatchEvent(new Event('input', { bubbles: true }));
        },

        /**
         * Set the text color
         */
        setTextColor(color) {
            const editor = document.getElementById('editor-content');

            // Try to restore saved selection
            this.restoreEditorSelection();

            const selection = window.getSelection();
            if (!selection.rangeCount) {
                if (editor) editor.focus();
                console.warn('No selection to apply color to');
                return;
            }

            // Check if selection is within editor
            const range = selection.getRangeAt(0);
            let node = range.commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;
            if (editor && !editor.contains(node)) {
                console.warn('Selection is not within editor');
                return;
            }

            document.execCommand('foreColor', false, color);
            this.currentTextColor = color;
            this.customColor = color;
            this.updateFormattingState();
            this.trackEditorSelection();

            if (editor) editor.dispatchEvent(new Event('input', { bubbles: true }));
        },

        /**
         * Update formatting state based on current selection
         */
        updateFormattingState() {
            const editor = document.getElementById('editor-content');
            const selection = window.getSelection();
            if (!editor || !selection.rangeCount) return;

            let node = selection.getRangeAt(0).commonAncestorContainer;
            if (node.nodeType === 3) node = node.parentNode;

            if (!editor.contains(node)) return;

            // Basic formatting
            this.isBold = document.queryCommandState('bold');
            this.isItalic = document.queryCommandState('italic');
            this.isUnderline = document.queryCommandState('underline');

            // Alignment
            if (document.queryCommandState('justifyLeft')) this.alignment = 'left';
            else if (document.queryCommandState('justifyCenter')) this.alignment = 'center';
            else if (document.queryCommandState('justifyRight')) this.alignment = 'right';
            else if (document.queryCommandState('justifyFull')) this.alignment = 'justify';

            // Current heading
            const block = document.queryCommandValue('formatBlock');
            const headingMap = {
                'h1': 'Heading 1',
                'h2': 'Heading 2',
                'h3': 'Heading 3',
                'h4': 'Heading 4',
                'p': 'Normal',
                '': 'Normal'
            };
            this.currentHeading = headingMap[block.toLowerCase()] || 'Normal';

            // Current font size
            const fontSizeSelection = window.getSelection();
            if (fontSizeSelection.rangeCount > 0) {
                const range = fontSizeSelection.getRangeAt(0);
                let node = range.commonAncestorContainer;
                if (node.nodeType === 3) node = node.parentNode;

                const computedStyle = window.getComputedStyle(node);
                const fontSizePx = parseFloat(computedStyle.fontSize);
                this.currentFontSize = Math.round(fontSizePx * 0.75); // px to pt
            }

            // Current text color
            const color = document.queryCommandValue('foreColor');
            if (color) {
                this.currentTextColor = this.rgbToHex(color);
                this.customColor = this.currentTextColor;
            }
        },

        /**
         * Convert RGB color string to hex format
         */
        rgbToHex(color) {
            if (!color || typeof color !== 'string') return '#000000';
            if (color.startsWith('#')) return color;

            const match = color.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)/);
            if (match) {
                const r = parseInt(match[1]).toString(16).padStart(2, '0');
                const g = parseInt(match[2]).toString(16).padStart(2, '0');
                const b = parseInt(match[3]).toString(16).padStart(2, '0');
                return '#' + r + g + b;
            }
            
            return '#000000';
        }
    };
}
