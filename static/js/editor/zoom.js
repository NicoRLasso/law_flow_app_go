/**
 * Zoom Behavior
 * Handles scaling of the template editor canvas
 */
function createZoomBehavior() {
    return {
        zoomLevel: 1.0,

        setZoom(level) {
            // Constrain between 0.3 (30%) and 2.0 (200%)
            this.zoomLevel = Math.min(Math.max(level, 0.30), 2.0);
            
            // Re-render page breaks as zoom might affect layout calculations
            // although transform scale usually doesn't affect flow, it's safer to re-check
            this.$nextTick(() => {
                this.updatePageBreaks();
            });
        },

        zoomIn() {
            this.setZoom(parseFloat((this.zoomLevel + 0.1).toFixed(1)));
        },

        zoomOut() {
            this.setZoom(parseFloat((this.zoomLevel - 0.1).toFixed(1)));
        },

        resetZoom() {
            this.setZoom(1.0);
        }
    };
}
