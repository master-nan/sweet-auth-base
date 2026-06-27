export type EnumDataType = {
    number: '[object Number]',
    string: '[object String]',
    boolean: '[object Boolean]',
    null: '[object Null]',
    undefined: '[object Undefined]',
    object: '[object Object]',
    array: '[object Array]',
    date: '[object Date]',
    regexp: '[object RegExp]',
    set: '[object Set]',
    map: '[object Map]',
    file: '[object File]'
}

export const DataType: EnumDataType = {
    number: '[object Number]',
    string: '[object String]',
    boolean: '[object Boolean]',
    null: '[object Null]',
    undefined: '[object Undefined]',
    object: '[object Object]',
    array: '[object Array]',
    date: '[object Date]',
    regexp: '[object RegExp]',
    set: '[object Set]',
    map: '[object Map]',
    file: '[object File]'
}

export function isNumber (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.number
}
export function isString (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.string
}
export function isBoolean (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.boolean
}
export function isNull (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.null
}
export function isUndefined (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.undefined
}
export function isObject (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.object
}
export function isArray (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.array
}
export function isDate (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.date
}
export function isRegExp (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.regexp
}
export function isSet (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.set
}
export function isMap (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.map
}
export function isFile (data: unknown) {
  return Object.prototype.toString.call(data) === DataType.file
}
