/**
 * Template Editor - Auto Paging Module
 * Handles automatic page break insertion for content that would overflow
 */

function createAutoPagingBehavior() {
    return {
        /**
         * Check and insert automatic page breaks where content would overflow
         */
        checkAutoPageBreaks() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;
            
            const selection = this.saveSelection();
            
            // PHASE 1: Reset - Remove existing auto breaks
            const existingAutoBreaks = editor.querySelectorAll('.auto-break');
            existingAutoBreaks.forEach(el => el.remove());
            
            // PHASE 2: Measure all children
            const children = Array.from(editor.children);
            const totalPageHeight = this.pageHeight + this.pageGap;
            
            const measurements = children.map(child => {
                if (child.classList.contains('page-break')) return null;
                if (child.style.display === 'none') return null;
                
                return {
                    el: child,
                    top: child.offsetTop,
                    height: child.offsetHeight,
                    prevIsManualBreak: child.previousElementSibling && 
                                     child.previousElementSibling.classList.contains('page-break') && 
                                     !child.previousElementSibling.classList.contains('auto-break')
                };
            }).filter(item => item !== null);

            // PHASE 3: Calculate which elements need breaks
            const insertions = [];
            for (const item of measurements) {
                // Skip very tall elements (>80% of page height)
                if (item.height > this.pageHeight * 0.8) continue;

                const pageIndex = Math.floor(item.top / totalPageHeight);
                const pageStart = pageIndex * totalPageHeight;
                const pageContentLimit = pageStart + this.pageHeight - this.pagePadding;

                // If element crosses page boundary
                if (item.top < pageContentLimit && (item.top + item.height) > pageContentLimit) {
                     if (item.prevIsManualBreak) continue;
                     insertions.push(item.el);
                }
            }

            // PHASE 4: Apply auto breaks
            if (insertions.length > 0) {
                 insertions.forEach(targetEl => {
                     const br = document.createElement('div');
                     br.className = 'page-break auto-break';
                     br.contentEditable = "false";
                     br.style.cssText = 'display: block; width: 100%; border-top: 1px dotted #ccc; margin: 0; opacity: 0.5;';
                     br.innerHTML = '<span style="position: absolute; right: 0; font-size: 8px; color: #999; pointer-events: none; padding-right: 4px;">Auto</span>';
                     editor.insertBefore(br, targetEl);
                 });

                 this.updatePageBreaks();
                 this.updatePageCount();
                 this.restoreSelection(selection);
            } else if (existingAutoBreaks.length > 0) {
                this.updatePageBreaks();
                this.updatePageCount();
                this.restoreSelection(selection);
            }
        }
    };
}
