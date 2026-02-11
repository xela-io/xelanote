import type { EditorView } from '@codemirror/view';

import type { AIAction } from '$lib/api';
import { type AITransformState,applyAITransform, prepareAITransform } from '$lib/editor/ai-actions';

interface OpenAIActionsMenuParams {
	rect: DOMRect;
	aiEnabled: boolean;
	setTriggerRect: (rect: DOMRect) => void;
	setOpen: (value: boolean) => void;
	showDisabledError: () => void;
}

export function openAIActionsMenuAction(params: OpenAIActionsMenuParams): void {
	const { rect, aiEnabled, setTriggerRect, setOpen, showDisabledError } = params;
	if (!aiEnabled) {
		showDisabledError();
		return;
	}

	setTriggerRect(rect);
	setOpen(true);
}

interface HandleAIActionSelectParams {
	action: AIAction;
	customPrompt?: string;
	currentContent: string;
	editorView: EditorView;
	setDropdownOpen: (value: boolean) => void;
	setDialogOpen: (value: boolean) => void;
	setTransformState: (state: AITransformState | null) => void;
	showTooShortError: () => void;
}

export async function handleAIActionSelectAction(params: HandleAIActionSelectParams): Promise<void> {
	const {
		action,
		customPrompt,
		currentContent,
		editorView,
		setDropdownOpen,
		setDialogOpen,
		setTransformState,
		showTooShortError,
	} = params;

	setDropdownOpen(false);

	await prepareAITransform(action, customPrompt, {
		getCurrentContent: () => currentContent,
		getEditorView: () => editorView,
		setDialogOpen,
		setTransformState,
		showError: showTooShortError,
	});
}

interface ApplyAITransformParams {
	transformedText: string;
	editorView: EditorView | undefined;
	aiTransformState: AITransformState | null;
	scheduleAutoSave: () => void;
	setDialogOpen: (value: boolean) => void;
	setTransformState: (state: AITransformState | null) => void;
	showSuccess: () => void;
}

export function applyAITransformAction(params: ApplyAITransformParams): void {
	const {
		transformedText,
		editorView,
		aiTransformState,
		scheduleAutoSave,
		setDialogOpen,
		setTransformState,
		showSuccess,
	} = params;

	applyAITransform(editorView, aiTransformState, transformedText);
	scheduleAutoSave();
	setDialogOpen(false);
	setTransformState(null);
	showSuccess();
}
