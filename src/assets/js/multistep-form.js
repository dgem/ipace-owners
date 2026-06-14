/**
 * multistep-form.js — Generic multi-step form controller.
 *
 * Works entirely from data attributes — not hardcoded to any single form.
 * Attaches to any element with [data-multistep-form].
 *
 * HTML structure expected:
 *   <form data-multistep-form>
 *     <div class="progress-indicator" data-progress>…</div>
 *     <div data-step="1">…step content…</div>
 *     <div data-step="2">…</div>
 *     …
 *     <div class="step-nav">
 *       <button type="button" data-prev>Back</button>
 *       <button type="button" data-next>Next</button>
 *       <button type="submit" data-submit>Submit</button>
 *     </div>
 *   </form>
 *
 * Progressive enhancement: without JS all steps remain visible in a
 * single-page layout. With JS, steps are revealed one at a time.
 */

(function () {
  'use strict';

  var REDUCED_MOTION = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  function initMultiStepForm(form) {
    var steps = Array.from(form.querySelectorAll('[data-step]'));
    if (!steps.length) return;

    var total       = steps.length;
    var currentStep = 0; // 0-indexed

    // Progress bar elements
    var progressFill = form.querySelector('[data-progress-fill]');
    var progressText = form.querySelector('[data-progress-text]');
    var progressStep = form.querySelector('[data-progress-step]');

    // Nav buttons
    var prevBtn   = form.querySelector('[data-prev]');
    var nextBtn   = form.querySelector('[data-next]');
    var submitBtn = form.querySelector('[data-submit]');

    // Result area
    var resultEl = form.querySelector('[data-submit-result]');

    // ── Initial state (JS enabled) ──────────────────────────────────────────
    steps.forEach(function (step, i) {
      if (i !== 0) {
        step.classList.add('step--hidden');
        // Disable focusable elements in hidden steps
        setInertState(step, true);
      }
    });

    updateUI();

    // ── Navigation handlers ─────────────────────────────────────────────────
    if (prevBtn) {
      prevBtn.addEventListener('click', function () {
        if (currentStep > 0) {
          goToStep(currentStep - 1);
        }
      });
    }

    if (nextBtn) {
      nextBtn.addEventListener('click', function () {
        if (validateCurrentStep() && currentStep < total - 1) {
          goToStep(currentStep + 1);
        }
      });
    }

    // Prevent real submission; show placeholder message instead
    form.addEventListener('submit', function (e) {
      e.preventDefault();

      if (!validateCurrentStep()) return;

      if (resultEl) {
        resultEl.classList.add('is-visible');
        resultEl.focus();
        // Hide step nav
        var nav = form.querySelector('.step-nav');
        if (nav) nav.style.display = 'none';
        // Hide all steps
        steps.forEach(function (s) {
          s.classList.add('step--hidden');
          setInertState(s, true);
        });
      } else {
        alert('Submission storage is not enabled yet. Your data has not been sent.');
      }
    });

    // ── Core: go to step ────────────────────────────────────────────────────
    function goToStep(index) {
      if (index < 0 || index >= total) return;

      // Hide current
      steps[currentStep].classList.add('step--hidden');
      setInertState(steps[currentStep], true);

      currentStep = index;

      // Show next
      steps[currentStep].classList.remove('step--hidden');
      setInertState(steps[currentStep], false);

      updateUI();
      moveFocusToStepHeading(steps[currentStep]);

      // Scroll to top of form (respects reduced motion)
      if (!REDUCED_MOTION) {
        form.scrollIntoView({ behavior: 'smooth', block: 'start' });
      } else {
        form.scrollIntoView({ block: 'start' });
      }
    }

    // ── Update progress & buttons ────────────────────────────────────────────
    function updateUI() {
      var humanStep  = currentStep + 1;
      var pct        = Math.round((humanStep / total) * 100);

      if (progressFill) {
        progressFill.style.width = pct + '%';
        progressFill.setAttribute('aria-valuenow', String(pct));
      }
      if (progressText) {
        progressText.textContent = 'Step ' + humanStep + ' of ' + total;
      }
      if (progressStep) {
        progressStep.textContent = humanStep;
      }

      // Prev button
      if (prevBtn) {
        prevBtn.disabled = currentStep === 0;
        prevBtn.style.visibility = currentStep === 0 ? 'hidden' : '';
      }

      // Next vs Submit
      var isLast = currentStep === total - 1;
      if (nextBtn)   nextBtn.style.display   = isLast ? 'none' : '';
      if (submitBtn) submitBtn.style.display = isLast ? '' : 'none';
    }

    // ── Validation (light, accessible) ─────────────────────────────────────
    function validateCurrentStep() {
      var step = steps[currentStep];
      var inputs = Array.from(step.querySelectorAll('input, select, textarea'));
      var valid = true;

      inputs.forEach(function (input) {
        // Clear previous error
        input.removeAttribute('aria-invalid');
        var errorEl = input.parentNode.querySelector('[role="alert"]') ||
                      document.getElementById(input.getAttribute('aria-describedby'));

        if (input.required && !input.value.trim()) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) errorEl.textContent = 'This field is required.';
          if (valid) {
            // Focus the first invalid field
            input.focus();
          }
          valid = false;
        } else if (input.type === 'email' && input.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.value)) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) errorEl.textContent = 'Please enter a valid email address.';
          if (valid) input.focus();
          valid = false;
        } else {
          if (errorEl) errorEl.textContent = '';
        }
      });

      return valid;
    }

    // ── Accessibility helpers ────────────────────────────────────────────────
    function moveFocusToStepHeading(stepEl) {
      var heading = stepEl.querySelector('[data-step-heading], h2, h3');
      if (heading) {
        heading.setAttribute('tabindex', '-1');
        heading.focus({ preventScroll: true });
      }
    }

    // Disable / enable focusable elements in a hidden step to prevent
    // keyboard users from tabbing into off-screen content.
    function setInertState(stepEl, inert) {
      var focusable = stepEl.querySelectorAll(
        'input, select, textarea, button, a[href]'
      );
      focusable.forEach(function (el) {
        if (inert) {
          el.setAttribute('tabindex', '-1');
        } else {
          el.removeAttribute('tabindex');
        }
      });
    }
  }

  // ── Bootstrap all forms on the page ────────────────────────────────────────
  document.querySelectorAll('[data-multistep-form]').forEach(initMultiStepForm);

})();
