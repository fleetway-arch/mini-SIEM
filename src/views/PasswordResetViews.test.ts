import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { setActivePinia, createPinia } from 'pinia'
import * as api from '../api/client'
import { useToastStore } from '../stores/toast'
import ForgotPasswordView from './ForgotPasswordView.vue'
import ResetPasswordView from './ResetPasswordView.vue'

vi.mock('../api/client', () => ({
  forgotPassword: vi.fn(),
  resetPassword: vi.fn(),
}))

const mocked = vi.mocked(api)
const Stub = { template: '<div />' }

async function mountAt(path: string, component: typeof ForgotPasswordView) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', name: 'login', component: Stub },
      { path: '/forgot-password', component },
      { path: '/reset-password/:token', component },
    ],
  })
  await router.push(path)
  const wrapper = mount(component, { global: { plugins: [router] } })
  return { wrapper, router }
}

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
})

describe('ForgotPasswordView', () => {
  it('validates the email before calling the API', async () => {
    const { wrapper } = await mountAt('/forgot-password', ForgotPasswordView)
    await wrapper.find('input[type="email"]').setValue('not-an-email')
    await wrapper.find('form').trigger('submit')

    expect(mocked.forgotPassword).not.toHaveBeenCalled()
    expect(wrapper.find('[role="alert"]').text()).toContain('Adresse email invalide')
  })

  it('shows the same confirmation whatever the server knows about the email', async () => {
    const texts: string[] = []
    for (const email of ['owner@acme.test', 'nobody@acme.test']) {
      mocked.forgotPassword.mockResolvedValue({ message: 'ignored by the UI' })
      const { wrapper } = await mountAt('/forgot-password', ForgotPasswordView)
      await wrapper.find('input[type="email"]').setValue(` ${email.toUpperCase()} `)
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(mocked.forgotPassword).toHaveBeenLastCalledWith(email)
      expect(wrapper.find('form').exists()).toBe(false)
      texts.push(wrapper.find('[role="status"]').text())
    }
    expect(texts[0]).toContain('Si un compte existe')
    expect(texts[0]).toBe(texts[1])
  })

  it('shows API errors and re-enables the button', async () => {
    mocked.forgotPassword.mockRejectedValue(new Error('Trop de tentatives. Réessayez dans 30 s.'))
    const { wrapper } = await mountAt('/forgot-password', ForgotPasswordView)
    await wrapper.find('input[type="email"]').setValue('owner@acme.test')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('[role="alert"]').text()).toContain('30 s')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeUndefined()
  })
})

describe('ResetPasswordView', () => {
  async function fill(wrapper: Awaited<ReturnType<typeof mountAt>>['wrapper'], password: string, confirm = password) {
    await wrapper.find('#reset-password').setValue(password)
    await wrapper.find('#reset-confirm').setValue(confirm)
    await wrapper.find('form').trigger('submit')
    await flushPromises()
  }

  it('rejects weak or mismatched passwords locally', async () => {
    const { wrapper } = await mountAt('/reset-password/tok123', ResetPasswordView)
    await fill(wrapper, 'weak', 'other')

    expect(mocked.resetPassword).not.toHaveBeenCalled()
    expect(wrapper.find('#reset-password-error').text()).toContain('règles')
    expect(wrapper.find('#reset-confirm-error').text()).toContain('ne correspondent pas')
  })

  it('resets the password, then redirects to /login with a success toast', async () => {
    mocked.resetPassword.mockResolvedValue({ message: 'password updated' })
    const { wrapper, router } = await mountAt('/reset-password/tok123', ResetPasswordView)
    await fill(wrapper, 'N3w!Passw0rd-x')

    expect(mocked.resetPassword).toHaveBeenCalledWith('tok123', 'N3w!Passw0rd-x')
    expect(router.currentRoute.value.name).toBe('login')
    const { toasts } = useToastStore()
    const toast = toasts[toasts.length - 1]
    expect(toast?.type).toBe('success')
  })

  it('shows the invalid-link state for an expired or used token', async () => {
    mocked.resetPassword.mockRejectedValue(new Error('invalid or expired reset link'))
    const { wrapper } = await mountAt('/reset-password/old', ResetPasswordView)
    await fill(wrapper, 'N3w!Passw0rd-x')

    expect(wrapper.text()).toContain('invalide, expiré ou a déjà été utilisé')
    expect(wrapper.find('a[href="/forgot-password"]').exists()).toBe(true)
  })

  it('keeps the form usable after another error', async () => {
    mocked.resetPassword.mockRejectedValue(new Error('La requête a expiré.'))
    const { wrapper } = await mountAt('/reset-password/tok123', ResetPasswordView)
    await fill(wrapper, 'N3w!Passw0rd-x')

    expect(wrapper.find('[role="alert"]').text()).toContain('La requête a expiré.')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeUndefined()
  })
})
