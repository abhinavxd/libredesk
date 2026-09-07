import countries from '@countries'

export const countryCallingOptions = countries.map((country) => ({
    label: country.name,
    value: country.iso_2,
    emoji: country.emoji,
    calling_code: country.calling_code
}))

export const countryOptions = countries.map((country) => ({
    label: country.name,
    value: country.iso_2,
    emoji: country.emoji
}))

export default countries;
