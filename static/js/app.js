console.log('Law Flow App - Custom JavaScript loaded');

// HTMX event listeners for debugging and custom behavior
document.addEventListener('htmx:beforeRequest', function(event) {
    console.log('HTMX Request starting:', event.detail);
});

document.addEventListener('htmx:afterRequest', function(event) {
    console.log('HTMX Request completed:', event.detail);
});

document.addEventListener('htmx:responseError', function(event) {
    console.error('HTMX Error:', event.detail);
});

// Alpine.js global store (if needed)
document.addEventListener('alpine:init', () => {
    console.log('Alpine.js initialized');
});

/* ============================================
   KINETIC TYPOGRAPHY - SCROLL ANIMATIONS
   ============================================ */

// Hero Parallax Effect
function initHeroParallax() {
    const hero = document.querySelector('.hero-parallax');
    if (!hero) return;

    // Check for reduced motion preference
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReducedMotion) return;

    window.addEventListener('scroll', () => {
        const scrolled = window.pageYOffset;
        const heroHeight = hero.offsetHeight;
        const scrollPercent = scrolled / heroHeight;

        // Scale from 1.0 to 1.2 and fade out
        const scale = 1 + (scrollPercent * 0.2);
        const opacity = Math.max(0, 1 - (scrollPercent * 2));

        hero.style.transform = `scale(${scale})`;
        hero.style.opacity = opacity;
    });
}

// Intersection Observer for fade-in animations
function initFadeInObserver() {
    const fadeElements = document.querySelectorAll('.fade-in');
    if (fadeElements.length === 0) return;

    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -100px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
            }
        });
    }, observerOptions);

    fadeElements.forEach(element => observer.observe(element));
}

// Accordion functionality
function initAccordions() {
    const accordionTriggers = document.querySelectorAll('.accordion-trigger');
    
    accordionTriggers.forEach(trigger => {
        trigger.addEventListener('click', () => {
            const content = trigger.nextElementSibling;
            const isOpen = content.classList.contains('open');
            
            // Close all other accordions (optional - remove for multi-open)
            document.querySelectorAll('.accordion-content').forEach(item => {
                if (item !== content) {
                    item.classList.remove('open');
                }
            });
            
            // Toggle current accordion
            if (isOpen) {
                content.classList.remove('open');
                trigger.setAttribute('aria-expanded', 'false');
            } else {
                content.classList.add('open');
                trigger.setAttribute('aria-expanded', 'true');
            }
        });
    });
}

// Initialize all animations on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    initHeroParallax();
    initFadeInObserver();
    initAccordions();
    console.log('Kinetic Typography animations initialized');
});
