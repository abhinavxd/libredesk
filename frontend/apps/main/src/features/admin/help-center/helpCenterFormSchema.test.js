import { describe, test, expect } from 'vitest'
import { createHelpCenterFormSchema, createHelpCenterBasicsSchema } from './helpCenterFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createHelpCenterFormSchema(mockT)
const basicsSchema = createHelpCenterBasicsSchema(mockT)

const validForm = {
    name: 'Docs',
    slug: 'docs',
    page_title: 'Help center'
}

describe('Help Center Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            template: 'docs',
            meta_description: 'Everything about our product',
            custom_domain: 'https://help.example.com',
            custom_css: 'body { color: red }',
            custom_js: 'console.log(1)',
            default_locale: 'fr',
            allowed_locales: ['en', 'fr'],
            theme: {
                color: 'blue',
                logo_url: '/uploads/logo.png',
                favicon: 'https://cdn.example.com/f.ico',
                tagline: 'We are here',
                nav_links: [{ label: 'Home', url: '/' }],
                header: {
                    heading: 'Hi',
                    background_type: 'gradient',
                    background_color: '#fff',
                    gradient_from: '#000000',
                    gradient_to: '#ffffffcc',
                    background_image: '/uploads/bg.png',
                    text_color: '#123'
                },
                layout: { collections: 'grid', columns: 3, show_popular_articles: true, popular_articles_label: 'Popular' },
                cards: { hide_description: false, hide_count: true, show_authors: true, show_icon_tile: true, icon_position: 'top' },
                announcement: { text: 'New release', link_label: 'Read', link_url: '/blog' },
                footer: { background_color: '#111111', text_color: '#eeeeee', tagline: 'Bye' },
                footer_links: [{ label: 'Terms', url: 'https://example.com/terms' }],
                social_links: [{ platform: 'x', url: 'https://x.com/example' }],
                article: { hide_toc: false, hide_related: true, show_author: true }
            }
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ ...validForm, name: '' })).toThrow()
    })

    test('name too long', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(201) })).toThrow()
    })

    test('name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(200) })).not.toThrow()
    })

    test('name length counts emoji as one character', () => {
        expect(() => schema.parse({ ...validForm, name: '😀'.repeat(200) })).not.toThrow()
    })

    test('slug missing', () => {
        const { slug, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('slug empty string', () => {
        expect(() => schema.parse({ ...validForm, slug: '' })).toThrow()
    })

    test('slug with uppercase rejected', () => {
        expect(() => schema.parse({ ...validForm, slug: 'Docs' })).toThrow()
    })

    test('slug with space rejected', () => {
        expect(() => schema.parse({ ...validForm, slug: 'my docs' })).toThrow()
    })

    test('slug with dash and underscore accepted', () => {
        expect(() => schema.parse({ ...validForm, slug: 'my-docs_2' })).not.toThrow()
    })

    test('slug too long', () => {
        expect(() => schema.parse({ ...validForm, slug: 'a'.repeat(201) })).toThrow()
    })

    test('page_title missing', () => {
        const { page_title, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('page_title empty string', () => {
        expect(() => schema.parse({ ...validForm, page_title: '' })).toThrow()
    })

    test('page_title too long', () => {
        expect(() => schema.parse({ ...validForm, page_title: 'a'.repeat(201) })).toThrow()
    })

    test('template defaults to classic', () => {
        expect(schema.parse(validForm).template).toBe('classic')
    })

    test('template invalid value', () => {
        expect(() => schema.parse({ ...validForm, template: 'minimal' })).toThrow()
    })

    test('meta_description too long', () => {
        expect(() => schema.parse({ ...validForm, meta_description: 'a'.repeat(501) })).toThrow()
    })

    test('meta_description at maximum length', () => {
        expect(() => schema.parse({ ...validForm, meta_description: 'a'.repeat(500) })).not.toThrow()
    })

    test('custom_domain without protocol rejected', () => {
        expect(() => schema.parse({ ...validForm, custom_domain: 'help.example.com' })).toThrow()
    })

    test('custom_domain with a path rejected', () => {
        expect(() => schema.parse({ ...validForm, custom_domain: 'https://help.example.com/docs' })).toThrow()
    })

    test('custom_domain empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, custom_domain: '' })).not.toThrow()
    })

    test('default_locale defaults to en', () => {
        expect(schema.parse(validForm).default_locale).toBe('en')
    })

    test('allowed_locales defaults to en', () => {
        expect(schema.parse(validForm).allowed_locales).toEqual(['en'])
    })

    test('default_locale must be in allowed_locales', () => {
        expect(() => schema.parse({ ...validForm, default_locale: 'fr', allowed_locales: ['en'] })).toThrow()
    })

    test('invalid locale format', () => {
        expect(() => schema.parse({ ...validForm, default_locale: 'english', allowed_locales: ['english'] })).toThrow()
    })

    test('region locale accepted', () => {
        expect(() => schema.parse({ ...validForm, default_locale: 'pt-BR', allowed_locales: ['pt-BR'] })).not.toThrow()
    })

    test('allowed_locales empty array', () => {
        expect(() => schema.parse({ ...validForm, allowed_locales: [] })).toThrow()
    })

    test('theme logo_url protocol relative rejected', () => {
        expect(() => schema.parse({ ...validForm, theme: { logo_url: '//cdn.example.com/l.png' } })).toThrow()
    })

    test('theme logo_url root relative accepted', () => {
        expect(() => schema.parse({ ...validForm, theme: { logo_url: '/uploads/l.png' } })).not.toThrow()
    })

    test('theme logo_url empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, theme: { logo_url: '' } })).not.toThrow()
    })

    test('theme header background_color invalid hex', () => {
        expect(() => schema.parse({ ...validForm, theme: { header: { background_color: 'red' } } })).toThrow()
    })

    test('theme header background_color accepts 3, 4, 6 and 8 digit hex', () => {
        for (const background_color of ['#fff', '#fff0', '#ffffff', '#ffffff00']) {
            expect(() => schema.parse({ ...validForm, theme: { header: { background_color } } })).not.toThrow()
        }
    })

    test('nav link requires label and url', () => {
        expect(() => schema.parse({ ...validForm, theme: { nav_links: [{ label: '', url: '/' }] } })).toThrow()
        expect(() => schema.parse({ ...validForm, theme: { nav_links: [{ label: 'Home', url: '' }] } })).toThrow()
        expect(() => schema.parse({ ...validForm, theme: { nav_links: [{ label: 'Home', url: 'home' }] } })).toThrow()
    })

    test('social link requires platform and url', () => {
        expect(() => schema.parse({ ...validForm, theme: { social_links: [{ platform: '', url: 'https://x.com/a' }] } })).toThrow()
        expect(() => schema.parse({ ...validForm, theme: { social_links: [{ platform: 'x', url: 'x.com/a' }] } })).toThrow()
    })

    test('card icon_position invalid value', () => {
        expect(() => schema.parse({ ...validForm, theme: { cards: { icon_position: 'bottom' } } })).toThrow()
    })

    test('announcement link without text rejected', () => {
        expect(() => schema.parse({
            ...validForm,
            theme: { announcement: { text: '', link_label: 'Read', link_url: '/blog' } }
        })).toThrow()
    })

    test('announcement text with only whitespace rejected when a link is set', () => {
        expect(() => schema.parse({
            ...validForm,
            theme: { announcement: { text: '   ', link_url: '/blog' } }
        })).toThrow()
    })

    test('announcement with no text and no link accepted', () => {
        expect(() => schema.parse({ ...validForm, theme: { announcement: {} } })).not.toThrow()
    })

    test('announcement with text and link accepted', () => {
        expect(() => schema.parse({
            ...validForm,
            theme: { announcement: { text: 'Hi', link_label: 'Read', link_url: '/blog' } }
        })).not.toThrow()
    })

    test('theme optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})

describe('Help Center Basics Schema', () => {
    test('valid form', () => {
        expect(() => basicsSchema.parse(validForm)).not.toThrow()
    })

    test('drops the fields outside basics', () => {
        const parsed = basicsSchema.parse({ ...validForm, custom_css: 'body{}' })
        expect(parsed).toEqual({ name: 'Docs', slug: 'docs', page_title: 'Help center', template: 'classic' })
    })

    test('name still required', () => {
        const { name, ...form } = validForm
        expect(() => basicsSchema.parse(form)).toThrow()
    })

    test('slug still validated', () => {
        expect(() => basicsSchema.parse({ ...validForm, slug: 'Bad Slug' })).toThrow()
    })

    test('locales not required', () => {
        expect(basicsSchema.parse(validForm).default_locale).toBeUndefined()
    })
})
