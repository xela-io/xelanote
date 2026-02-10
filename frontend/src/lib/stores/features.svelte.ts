// Feature detection store using Svelte 5 runes
// Detects which optional features are enabled on the backend

import { getApiBaseUrl } from '$lib/config';
import { getFeature, setFeature } from '$lib/api';

// === Graph Feature (Backend Detection) ===
let graphFeatureEnabled = $state(false);
let featureCheckDone = $state(false);

export function getGraphFeatureEnabled() {
  return graphFeatureEnabled;
}

export function getFeatureCheckDone() {
  return featureCheckDone;
}

// Detect if graph feature is enabled by checking if endpoint exists
export async function detectGraphFeature() {
  if (typeof fetch === 'undefined') {
    // Server-side rendering - skip detection
    featureCheckDone = true;
    return;
  }

  try {
    // HEAD request to check if endpoint exists (returns 404 if feature flag off)
    const apiBase = getApiBaseUrl();
    const response = await fetch(`${apiBase}/graph`, {
      method: 'HEAD',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // If status is not 404, the endpoint exists (feature is enabled)
    // Note: We might get 401 (unauthorized) if not logged in, but that still means the feature exists
    graphFeatureEnabled = response.status !== 404;
  } catch (error) {
    console.error('Failed to detect graph feature:', error);
    graphFeatureEnabled = false;
  } finally {
    featureCheckDone = true;
  }
}

// === Journal Feature (User-Specific Toggle) ===
let journalFeatureEnabled = $state(false);
let journalFeatureLoading = $state(false);
let journalFeatureLoaded = $state(false);

export function getJournalFeatureEnabled() {
  return journalFeatureEnabled;
}

export function getJournalFeatureLoading() {
  return journalFeatureLoading;
}

export function getJournalFeatureLoaded() {
  return journalFeatureLoaded;
}

/**
 * Load the journal feature setting for the current user.
 * Should be called after authentication.
 */
export async function loadJournalFeature() {
  if (journalFeatureLoading) return;

  journalFeatureLoading = true;
  try {
    const feature = await getFeature('journal');
    journalFeatureEnabled = feature.enabled;
    journalFeatureLoaded = true;
  } catch (error) {
    console.error('Failed to load journal feature:', error);
    journalFeatureEnabled = false;
    journalFeatureLoaded = true;
  } finally {
    journalFeatureLoading = false;
  }
}

/**
 * Toggle the journal feature for the current user.
 */
export async function toggleJournalFeature(enabled: boolean) {
  journalFeatureLoading = true;
  try {
    await setFeature('journal', enabled);
    journalFeatureEnabled = enabled;
  } catch (error) {
    console.error('Failed to toggle journal feature:', error);
    throw error;
  } finally {
    journalFeatureLoading = false;
  }
}

/**
 * Reset journal feature state (called on logout).
 */
export function resetJournalFeature() {
  journalFeatureEnabled = false;
  journalFeatureLoaded = false;
  journalFeatureLoading = false;
}

// === Recipe Feature (User-Specific Toggle) ===
let recipeFeatureEnabled = $state(false);
let recipeFeatureLoading = $state(false);
let recipeFeatureLoaded = $state(false);

export function getRecipeFeatureEnabled() {
  return recipeFeatureEnabled;
}

export function getRecipeFeatureLoading() {
  return recipeFeatureLoading;
}

export function getRecipeFeatureLoaded() {
  return recipeFeatureLoaded;
}

/**
 * Load the recipe feature setting for the current user.
 * Should be called after authentication.
 */
export async function loadRecipeFeature() {
  if (recipeFeatureLoading) return;

  recipeFeatureLoading = true;
  try {
    const feature = await getFeature('recipe');
    recipeFeatureEnabled = feature.enabled;
    recipeFeatureLoaded = true;
  } catch (error) {
    console.error('Failed to load recipe feature:', error);
    recipeFeatureEnabled = false;
    recipeFeatureLoaded = true;
  } finally {
    recipeFeatureLoading = false;
  }
}

/**
 * Toggle the recipe feature for the current user.
 */
export async function toggleRecipeFeature(enabled: boolean) {
  recipeFeatureLoading = true;
  try {
    await setFeature('recipe', enabled);
    recipeFeatureEnabled = enabled;
  } catch (error) {
    console.error('Failed to toggle recipe feature:', error);
    throw error;
  } finally {
    recipeFeatureLoading = false;
  }
}

/**
 * Reset recipe feature state (called on logout).
 */
export function resetRecipeFeature() {
  recipeFeatureEnabled = false;
  recipeFeatureLoaded = false;
  recipeFeatureLoading = false;
}
