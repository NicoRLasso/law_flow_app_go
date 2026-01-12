/**
 * Kinetic Typography - Marquee Component
 * GPU-accelerated infinite scrolling marquee
 */

class Marquee {
  constructor(element, options = {}) {
    this.element = element;
    this.content = element.querySelector('.marquee-content');
    this.speed = options.speed || 20; // seconds for one loop
    this.direction = options.direction || 'left';
    this.pauseOnHover = options.pauseOnHover || false;
    
    this.init();
  }

  init() {
    // Check for reduced motion preference
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    
    if (prefersReducedMotion) {
      this.content.style.animation = 'none';
      return;
    }

    // Duplicate content for seamless loop
    this.duplicateContent();
    
    // Set animation duration
    this.content.style.animationDuration = `${this.speed}s`;
    
    // Set direction
    if (this.direction === 'right') {
      this.content.style.animationDirection = 'reverse';
    }

    // Pause on hover if enabled
    if (this.pauseOnHover) {
      this.element.addEventListener('mouseenter', () => {
        this.content.style.animationPlayState = 'paused';
      });
      
      this.element.addEventListener('mouseleave', () => {
        this.content.style.animationPlayState = 'running';
      });
    }
  }

  duplicateContent() {
    // Clone all children and append to create seamless loop
    const items = Array.from(this.content.children);
    items.forEach(item => {
      const clone = item.cloneNode(true);
      this.content.appendChild(clone);
    });
  }

  destroy() {
    this.content.style.animation = 'none';
  }
}

// Auto-initialize all marquees on page load
document.addEventListener('DOMContentLoaded', () => {
  const marquees = document.querySelectorAll('.marquee');
  
  marquees.forEach(marquee => {
    const speed = marquee.classList.contains('marquee-fast') ? 15 :
                  marquee.classList.contains('marquee-slow') ? 30 : 20;
    
    new Marquee(marquee, { speed });
  });
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = Marquee;
}
