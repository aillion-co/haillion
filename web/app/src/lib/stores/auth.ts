import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export interface User {
	id: string;
	email: string;
	role: 'rider' | 'driver';
}

const STORAGE_KEY = 'aillion_user';

const initialUser = browser ? JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null') : null;

export const authStore = writable<User | null>(initialUser);

if (browser) {
	authStore.subscribe((user) => {
		if (user) {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(user));
		} else {
			localStorage.removeItem(STORAGE_KEY);
		}
	});
}
