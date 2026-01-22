/* ============================================
   KINETIC TYPOGRAPHY - SCROLL ANIMATIONS
   ============================================ */

// Check for reduced motion preference globally
const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// Hero Parallax Effect
function initHeroParallax() {
    const hero = document.querySelector('.hero-parallax');
    if (!hero) return;

    if (prefersReducedMotion) return;

    let heroHeight = hero.offsetHeight;
    let ticking = false;

    window.addEventListener('resize', () => {
        heroHeight = hero.offsetHeight;
    });

    window.addEventListener('scroll', () => {
        if (!ticking) {
            window.requestAnimationFrame(() => {
                const scrolled = window.window.pageYOffset || document.documentElement.scrollTop;
                const scrollPercent = Math.min(1, Math.max(0, scrolled / heroHeight));

                // Scale from 1.0 to 1.2 and fade out
                const scale = 1 + (scrollPercent * 0.2);
                const opacity = Math.max(0, 1 - (scrollPercent * 2));

                hero.style.transform = `scale(${scale})`;
                hero.style.opacity = opacity;
                
                ticking = false;
            });
            ticking = true;
        }
    }, { passive: true });
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

/* ============================================
   SCROLL-TRIGGERED REVEAL ANIMATIONS
   ============================================ */

function initScrollReveal() {
    const revealElements = document.querySelectorAll('.reveal, .reveal-left, .reveal-right, .reveal-scale');
    if (revealElements.length === 0) return;

    // Skip animations if user prefers reduced motion
    if (prefersReducedMotion) {
        revealElements.forEach(el => el.classList.add('visible'));
        return;
    }

    const observerOptions = {
        threshold: 0.15,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
                // Optionally unobserve after animation (one-time reveal)
                observer.unobserve(entry.target);
            }
        });
    }, observerOptions);

    revealElements.forEach(element => observer.observe(element));
}

/* ============================================
   STICKY MOBILE CTA
   ============================================ */

function initStickyCTA() {
    const stickyCTA = document.querySelector('.sticky-cta-mobile');
    if (!stickyCTA) return;

    const showThreshold = 600; // Show after scrolling 600px
    let ticking = false;

    window.addEventListener('scroll', () => {
        if (!ticking) {
            window.requestAnimationFrame(() => {
                const currentScrollY = window.scrollY || document.documentElement.scrollTop;

                if (currentScrollY > showThreshold) {
                    stickyCTA.classList.add('visible');
                } else {
                    stickyCTA.classList.remove('visible');
                }
                ticking = false;
            });
            ticking = true;
        }
    }, { passive: true });
}

/* ============================================
   COUNTER ANIMATION (Alpine.js helper)
   ============================================ */

// This function can be called from Alpine.js x-init
window.animateCounter = function(element, target, duration = 2000) {
    if (prefersReducedMotion) {
        element.textContent = target.toLocaleString();
        return;
    }

    const start = 0;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);

        // Easing function (ease-out)
        const easeOut = 1 - Math.pow(1 - progress, 3);
        const current = Math.floor(start + (target - start) * easeOut);

        element.textContent = current.toLocaleString();

        if (progress < 1) {
            requestAnimationFrame(update);
        } else {
            element.textContent = target.toLocaleString();
        }
    }

    requestAnimationFrame(update);
};

/* ============================================
   SMOOTH SCROLL FOR ANCHOR LINKS
   ============================================ */

function initSmoothScroll() {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            const targetId = this.getAttribute('href');
            if (targetId === '#') return;

            const target = document.querySelector(targetId);
            if (target) {
                e.preventDefault();
                target.scrollIntoView({
                    behavior: prefersReducedMotion ? 'auto' : 'smooth',
                    block: 'start'
                });
            }
        });
    });
}

/* ============================================
   ACCORDION FUNCTIONALITY
   ============================================ */

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

/* ============================================
   INITIALIZE ALL SCRIPTS
   ============================================ */

document.addEventListener('DOMContentLoaded', () => {
    initHeroParallax();
    initFadeInObserver();
    initScrollReveal();
    initStickyCTA();
    initSmoothScroll();
    initAccordions();

    // Toast Notification Handler
    document.body.addEventListener('show-toast', function(evt) {
        const message = evt.detail.message || evt.detail.value || evt.detail;
        const type = evt.detail.type || 'success'; // success, error
        
        const toast = document.createElement('div');
        toast.className = `fixed top-4 right-4 z-[100] p-4 rounded-lg shadow-lg transform transition-all duration-300 translate-y-[-20px] opacity-0 ${
            type === 'error' ? 'bg-red-500 text-white' : 'bg-green-500 text-white'
        }`;
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        // Animate in
        requestAnimationFrame(() => {
            toast.classList.remove('translate-y-[-20px]', 'opacity-0');
        });
        
        // Remove after 3 seconds
        setTimeout(() => {
            toast.classList.add('translate-y-[-20px]', 'opacity-0');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    });

    // Modal Close Handler
    document.body.addEventListener('close-modal', function(evt) {
        const modalId = evt.detail.id || evt.detail.value || evt.detail;
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.remove();
        }
    });
});
